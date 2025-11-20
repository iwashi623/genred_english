import boto3
import os
import uuid
from datetime import datetime, timedelta
from fastapi import FastAPI, Depends, File, UploadFile, HTTPException, Path, Query, Response
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from pydantic_settings import BaseSettings
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from .database import get_db
from .models import Problem
from .models import Result
from .models import Genre
from .models import User

class Settings(BaseSettings):
    try_sound_bucket: str
    region_name: str

    class Config:
        env_file = ".env"


class LoginRequest(BaseModel):
    username: str


settings = Settings()
app = FastAPI()
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/")
async def root():
    return {"message": "Hello World"}


@app.post("/login")
async def login(
    request: LoginRequest,
    response: Response,
    db: AsyncSession = Depends(get_db)
):
    """
    ユーザー名でログイン処理を行う
    - ユーザーが存在すればそのIDを使用
    - 存在しなければ新規ユーザーを作成
    - UserIDをクッキーにセットして返す
    """
    # ユーザー名の検証
    if not request.username or not request.username.strip():
        raise HTTPException(
            status_code=400,
            detail="Username cannot be empty"
        )

    # ユーザー名でDBを検索
    result = await db.execute(
        select(User).where(User.name == request.username)
    )
    user = result.scalars().first()

    # ユーザーが存在しない場合は新規作成
    if not user:
        user = User(name=request.username)
        db.add(user)
        await db.commit()
        await db.refresh(user)

    # クッキーにUserIDをセット
    response.set_cookie(
        key="user_id",
        value=str(user.id),
        httponly=True,
        samesite="lax"
    )

    return {
        "user_id": user.id,
        "username": user.name,
        "message": "Login successful"
    }


@app.get("/problems")
async def get_problems(db: AsyncSession = Depends(get_db)):
    result = await db.execute(
        select(Problem)
        .order_by(Problem.created_at.desc())
        .limit(30)
    )
    problems = result.scalars().all()

    return {
        "problems": [
            {
                "id": problem.id,
                "genre_id": problem.genre_id,
                "text": problem.text,
                "answer_file_path": problem.answer_file_path,
                "created_at": problem.created_at.isoformat() if problem.created_at else None,
            }
            for problem in problems
        ]
    }

@app.get("/problems/{problem_id}/result")
async def get_latest_result(
    problem_id: int = Path(...),
    user_id: int = Query(...),
    db: AsyncSession = Depends(get_db),
):
    result = await db.execute(
        select(Result)
        .where(Result.problem_id == problem_id, Result.user_id == user_id)
        .order_by(Result.created_at.desc())
        .limit(1)
    )
    result = result.scalars().first()
    if not result:
        raise HTTPException(status_code=404, detail="result not found")

    return {
        "id": result.id,
        "user_id": result.user_id,
        "problem_id": result.problem_id,
        "score": float(result.score) if result.score is not None else None,
        "try_file_path": result.try_file_path,
        "created_at": result.created_at.isoformat() if result.created_at else None,
    }

@app.get("/genres")
async def get_genres(db: AsyncSession = Depends(get_db)):
    result = await db.execute(
        select(Genre)
    )
    genres = result.scalars().all()

    return {
        "genres": [
            {
                "id": genres.id,
                "name": genres.name,
                "display_name": genres.display_name,
                "created_at": genres.created_at.isoformat() if genres.created_at else None,
            }
            for genres in genres
        ]
    }

@app.post("/upload")
async def upload_try_sound(
    file: UploadFile = File(...)
):
    file_content = await file.read()
    basename, _ = os.path.splitext(file.filename)
    problem_id, user_id = basename.split("_")
    file_id = uuid.uuid1()
    s3 = boto3.client("s3", region_name=settings.region_name)

    try:
        _ = s3.put_object(
            Body=file_content,
            Bucket=settings.try_sound_bucket,
            Key=f"problems/{problem_id}/users/{user_id}/{file_id}.mp3"
        )
        return {"status": "ok", "file_id": str(file_id)}
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=f"Failed to upload file to S3: {str(e)}"
        )


@app.get("/ranking")
async def get_ranking(
    db: AsyncSession = Depends(get_db),
):
    # 直近1時間の時刻を計算
    one_hour_ago = datetime.now() - timedelta(hours=1)

    result = await db.execute(
        select(Result.score.label("score"), User.name.label("name"))
        .where(Result.created_at >= one_hour_ago)
        .order_by(Result.score.desc())
        .join(User, Result.user_id == User.id)
        .limit(10)
    )
    results = result.mappings().all()

    return {
        "ranking": [{
            "name": result.name,
            "score": float(result.score) if result.score is not None else None,
        } for result in results]
    }
