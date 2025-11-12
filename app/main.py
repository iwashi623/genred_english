import boto3
import os
import uuid
from fastapi import FastAPI, Depends, File, UploadFile, HTTPException, Path, Query
from fastapi.middleware.cors import CORSMiddleware
from pydantic_settings import BaseSettings
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from .database import get_db
from .models import Problem
from .models import Result

class Settings(BaseSettings):
    try_sound_bucket: str
    region_name: str

    class Config:
        env_file = ".env"


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
