package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	"github.com/aws/aws-sdk-go-v2/service/transcribe/types"
	_ "github.com/go-sql-driver/mysql"
)

var (
	db              *sql.DB
	transcribeClient *transcribe.Client
)

// 初期化処理
func init() {
	var err error

	// AWS設定の読み込み
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(fmt.Sprintf("failed to load AWS config: %v", err))
	}

	transcribeClient = transcribe.NewFromConfig(cfg)
}

// S3イベントハンドラー
func handler(ctx context.Context, s3Event events.S3Event) error {
	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key

		fmt.Printf("Processing file: s3://%s/%s\n", bucket, key)

		// パスから problem_id と user_id を抽出
		// パス形式: problems/{problem_id}/users/{user_id}/*.mp3
		problemID, userID, err := extractIDsFromPath(key)
		if err != nil {
			fmt.Printf("Failed to extract IDs from path: %v\n", err)
			continue
		}

		fmt.Printf("Extracted - problem_id: %d, user_id: %d\n", problemID, userID)

		// RDSから問題文を取得
		problemText, err := getProblemText(ctx, problemID)
		if err != nil {
			fmt.Printf("Failed to get problem text: %v\n", err)
			continue
		}

		fmt.Printf("Problem text retrieved: %s\n", problemText)

		// Transcribeで文字起こし
		transcribedText, err := transcribeAudio(ctx, bucket, key)
		if err != nil {
			fmt.Printf("Failed to transcribe audio: %v\n", err)
			continue
		}

		fmt.Printf("Transcribed text: %s\n", transcribedText)

		// 採点
		score := calculateScore(problemText, transcribedText)
		fmt.Printf("Calculated score: %.2f\n", score)

		// 結果をRDSに保存
		s3Path := fmt.Sprintf("s3://%s/%s", bucket, key)
		err = saveResult(ctx, userID, problemID, transcribedText, score, s3Path)
		if err != nil {
			fmt.Printf("Failed to save result: %v\n", err)
			continue
		}

		fmt.Printf("Result saved successfully\n")
	}

	return nil
}

// S3パスから problem_id と user_id を抽出
func extractIDsFromPath(path string) (problemID, userID int64, err error) {
	// パターン: problems/{problem_id}/users/{user_id}/*.mp3
	re := regexp.MustCompile(`problems/(\d+)/users/(\d+)/.*\.mp3$`)
	matches := re.FindStringSubmatch(path)

	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("invalid path format: %s", path)
	}

	problemID, err = strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid problem_id: %v", err)
	}

	userID, err = strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid user_id: %v", err)
	}

	return problemID, userID, nil
}

// RDSから問題文を取得
func getProblemText(ctx context.Context, problemID int64) (string, error) {
	if db == nil {
		if err := initDB(); err != nil {
			return "", err
		}
	}

	var text string
	query := "SELECT text FROM problems WHERE id = ?"
	err := db.QueryRowContext(ctx, query, problemID).Scan(&text)
	if err != nil {
		return "", fmt.Errorf("failed to query problem: %v", err)
	}

	return text, nil
}

// DB接続を初期化
func initDB() error {
	dbHost := getEnv("DB_HOST", "")
	dbUser := getEnv("DB_USER", "")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "")

	if dbHost == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		return fmt.Errorf("DB_HOST, DB_USER, DB_PASSWORD, DB_NAME environment variables must be set")
	}

	// DSN形式: user:password@tcp(host:port)/dbname?parseTime=true
	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbName)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// 接続テスト
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// 接続プール設定
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return nil
}

// 音声ファイルを文字起こし
func transcribeAudio(ctx context.Context, bucket, key string) (string, error) {
	// ジョブ名を生成（ユニークである必要がある）
	jobName := fmt.Sprintf("transcribe-%d", time.Now().UnixNano())

	// S3 URI
	mediaFileURI := fmt.Sprintf("s3://%s/%s", bucket, key)

	// Transcribeジョブを開始
	_, err := transcribeClient.StartTranscriptionJob(ctx, &transcribe.StartTranscriptionJobInput{
		TranscriptionJobName: aws.String(jobName),
		Media: &types.Media{
			MediaFileUri: aws.String(mediaFileURI),
		},
		MediaFormat:  types.MediaFormatMp3,
		LanguageCode: types.LanguageCodeEnUs, // 英語
	})
	if err != nil {
		return "", fmt.Errorf("failed to start transcription job: %v", err)
	}

	fmt.Printf("Started transcription job: %s\n", jobName)

	// ジョブの完了を待つ
	maxRetries := 60 // 最大5分待機（5秒間隔）
	for range maxRetries {
		time.Sleep(5 * time.Second)

		result, err := transcribeClient.GetTranscriptionJob(ctx, &transcribe.GetTranscriptionJobInput{
			TranscriptionJobName: aws.String(jobName),
		})
		if err != nil {
			return "", fmt.Errorf("failed to get transcription job: %v", err)
		}

		status := result.TranscriptionJob.TranscriptionJobStatus
		fmt.Printf("Transcription job status: %s\n", status)

		switch status {
		case types.TranscriptionJobStatusCompleted:
			// 結果を取得
			if result.TranscriptionJob.Transcript != nil && result.TranscriptionJob.Transcript.TranscriptFileUri != nil {
				transcriptText, err := fetchTranscriptText(ctx, *result.TranscriptionJob.Transcript.TranscriptFileUri)
				if err != nil {
					return "", err
				}

				// ジョブを削除（クリーンアップ）
				_, _ = transcribeClient.DeleteTranscriptionJob(ctx, &transcribe.DeleteTranscriptionJobInput{
					TranscriptionJobName: aws.String(jobName),
				})

				return transcriptText, nil
			}
			return "", fmt.Errorf("transcript URI is missing")
		case types.TranscriptionJobStatusFailed:
			// ジョブを削除
			_, _ = transcribeClient.DeleteTranscriptionJob(ctx, &transcribe.DeleteTranscriptionJobInput{
				TranscriptionJobName: aws.String(jobName),
			})

			failureReason := ""
			if result.TranscriptionJob.FailureReason != nil {
				failureReason = *result.TranscriptionJob.FailureReason
			}
			return "", fmt.Errorf("transcription job failed: %s", failureReason)
		}
	}

	return "", fmt.Errorf("transcription job timed out")
}

// Transcript結果を取得
func fetchTranscriptText(ctx context.Context, uri string) (string, error) {
	// AWS SDK v2を使ってTranscript JSONをダウンロード
	// 簡易実装: HTTPクライアントで直接取得
	// 実際のプロダクションコードでは S3 から直接取得する方が良い
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	// presigned URLとして扱う（TranscribeはpresignされたURLを返す）
	client := &transcribe.Client{}
	_ = client

	// 簡易実装: net/httpで取得
	// より堅牢な実装が必要な場合は、S3から直接ダウンロードする
	resp, err := getHTTPClient().Get(uri)
	if err != nil {
		return "", fmt.Errorf("failed to fetch transcript: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch transcript, status code: %d", resp.StatusCode)
	}

	// JSONをパース
	var transcriptResult struct {
		Results struct {
			Transcripts []struct {
				Transcript string `json:"transcript"`
			} `json:"transcripts"`
		} `json:"results"`
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&transcriptResult); err != nil {
		return "", fmt.Errorf("failed to decode transcript JSON: %v", err)
	}

	if len(transcriptResult.Results.Transcripts) == 0 {
		return "", fmt.Errorf("no transcripts found in result")
	}

	// 注: cfgは現在未使用だが、将来の拡張のために保持
	_ = cfg

	return transcriptResult.Results.Transcripts[0].Transcript, nil
}

// HTTPクライアントを取得
func getHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// 採点アルゴリズム（レーベンシュタイン距離ベース）
func calculateScore(original, transcribed string) float64 {
	// テキストの正規化
	original = normalizeText(original)
	transcribed = normalizeText(transcribed)

	// レーベンシュタイン距離を計算
	distance := levenshteinDistance(original, transcribed)

	// 最大長を取得
	maxLen := math.Max(float64(len(original)), float64(len(transcribed)))
	if maxLen == 0 {
		return 0.0
	}

	// 類似度を計算（100点満点）
	similarity := (1.0 - float64(distance)/maxLen) * 100.0

	// 0-100の範囲に制限
	if similarity < 0 {
		similarity = 0
	} else if similarity > 100 {
		similarity = 100
	}

	// 小数点第2位まで丸める
	return math.Round(similarity*100) / 100
}

// テキストの正規化
func normalizeText(text string) string {
	// 小文字に変換
	text = strings.ToLower(text)

	// 余分な空白を削除
	text = strings.TrimSpace(text)

	// 複数の空白を1つに
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return text
}

// レーベンシュタイン距離の計算
func levenshteinDistance(s1, s2 string) int {
	len1 := len(s1)
	len2 := len(s2)

	// DPテーブルを作成
	dp := make([][]int, len1+1)
	for i := range dp {
		dp[i] = make([]int, len2+1)
	}

	// 初期化
	for i := 0; i <= len1; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		dp[0][j] = j
	}

	// DP計算
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			dp[i][j] = min(
				dp[i-1][j]+1,      // 削除
				dp[i][j-1]+1,      // 挿入
				dp[i-1][j-1]+cost, // 置換
			)
		}
	}

	return dp[len1][len2]
}

// 最小値を返す
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// 結果をRDSに保存
func saveResult(ctx context.Context, userID, problemID int64, answeredText string, score float64, tryFilePath string) error {
	if db == nil {
		if err := initDB(); err != nil {
			return err
		}
	}

	query := `
		INSERT INTO results (user_id, problem_id, answered_text, score, try_file_path)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.ExecContext(ctx, query, userID, problemID, answeredText, score, tryFilePath)
	if err != nil {
		return fmt.Errorf("failed to insert result: %v", err)
	}

	return nil
}

// 環境変数を取得
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	lambda.Start(handler)
}
