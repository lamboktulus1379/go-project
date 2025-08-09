package persistence

import (
    "context"
    "database/sql"
    "strings"
    "time"

    "my-project/domain/model"
)

// ShareRepository implements sharing persistence using MySQL (native sql.DB)
type ShareRepository struct {
    db *sql.DB
}

func NewShareRepository(db *sql.DB) *ShareRepository { return &ShareRepository{db: db} }

func (r *ShareRepository) UpsertTrackShares(ctx context.Context, videoID, userID string, platforms []string, initialStatus string) ([]*model.VideoShareRecord, error) {
    out := make([]*model.VideoShareRecord, 0, len(platforms))
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil { return nil, err }
    defer func() { if err != nil { _ = tx.Rollback() } }()
    now := time.Now().UTC()
    for _, p := range platforms {
        p = strings.ToLower(p)
        // PostgreSQL upsert using ON CONFLICT
        q := `INSERT INTO video_share_records (video_id, platform, user_id, status, attempt_count, created_at, updated_at)
              VALUES ($1,$2,$3,$4,1,$5,$5)
              ON CONFLICT (video_id, platform, user_id) DO UPDATE SET
                attempt_count = video_share_records.attempt_count + CASE WHEN video_share_records.status <> EXCLUDED.status THEN 1 ELSE 0 END,
                status = CASE WHEN video_share_records.status = 'success' THEN video_share_records.status ELSE EXCLUDED.status END,
                updated_at = EXCLUDED.updated_at` 
        if _, err = tx.ExecContext(ctx, q, videoID, p, userID, initialStatus, now); err != nil { return nil, err }
        row := tx.QueryRowContext(ctx, `SELECT id, video_id, platform, user_id, status, error_message, attempt_count, created_at, updated_at FROM video_share_records WHERE video_id=$1 AND platform=$2 AND user_id=$3`, videoID, p, userID)
        rec := &model.VideoShareRecord{}
        var errMsg sql.NullString
        if err = row.Scan(&rec.ID, &rec.VideoID, &rec.Platform, &rec.UserID, &rec.Status, &errMsg, &rec.AttemptCount, &rec.CreatedAt, &rec.UpdatedAt); err != nil { return nil, err }
        if errMsg.Valid { rec.ErrorMessage = &errMsg.String }
        out = append(out, rec)
    }
    if err = tx.Commit(); err != nil { return nil, err }
    return out, nil
}

func (r *ShareRepository) GetShareStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error) {
    rows, err := r.db.QueryContext(ctx, `SELECT id, video_id, platform, user_id, status, error_message, attempt_count, created_at, updated_at FROM video_share_records WHERE video_id=$1 AND user_id=$2`, videoID, userID)
    if err != nil { return nil, err }
    defer rows.Close()
    var list []*model.VideoShareRecord
    for rows.Next() {
        rec := &model.VideoShareRecord{}
        var errMsg sql.NullString
        if err := rows.Scan(&rec.ID, &rec.VideoID, &rec.Platform, &rec.UserID, &rec.Status, &errMsg, &rec.AttemptCount, &rec.CreatedAt, &rec.UpdatedAt); err != nil { return nil, err }
        if errMsg.Valid { rec.ErrorMessage = &errMsg.String }
        list = append(list, rec)
    }
    return list, nil
}

func (r *ShareRepository) CreateAudit(ctx context.Context, audits []*model.VideoShareAudit) error {
    if len(audits) == 0 { return nil }
    q := `INSERT INTO video_share_audit (record_id, video_id, platform, user_id, status, error_message, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
    now := time.Now().UTC()
    for _, a := range audits {
        if a.CreatedAt.IsZero() { a.CreatedAt = now }
        _, err := r.db.ExecContext(ctx, q, a.RecordID, a.VideoID, a.Platform, a.UserID, a.Status, a.ErrorMessage, a.CreatedAt)
        if err != nil { return err }
    }
    return nil
}

func (r *ShareRepository) EnqueueJobs(ctx context.Context, records []*model.VideoShareRecord) error {
    if len(records) == 0 { return nil }
    q := `INSERT INTO share_jobs (record_id, platform, status, attempts, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$5)`
    now := time.Now().UTC()
    for _, rec := range records {
        if rec.Status != "pending" { continue }
        if _, err := r.db.ExecContext(ctx, q, rec.ID, rec.Platform, "pending", 0, now); err != nil { return err }
    }
    return nil
}

func (r *ShareRepository) FetchPendingJobs(ctx context.Context, limit int) ([]*model.ShareJob, error) {
    rows, err := r.db.QueryContext(ctx, `SELECT id, record_id, platform, status, attempts, last_error, created_at, updated_at FROM share_jobs WHERE status='pending' ORDER BY created_at ASC LIMIT $1`, limit)
    if err != nil { return nil, err }
    defer rows.Close()
    var jobs []*model.ShareJob
    for rows.Next() {
        j := &model.ShareJob{}
        var lastErr sql.NullString
        if err := rows.Scan(&j.ID, &j.RecordID, &j.Platform, &j.Status, &j.Attempts, &lastErr, &j.CreatedAt, &j.UpdatedAt); err != nil { return nil, err }
        if lastErr.Valid { j.LastError = &lastErr.String }
        jobs = append(jobs, j)
    }
    return jobs, nil
}

func (r *ShareRepository) MarkJobRunning(ctx context.Context, jobID int64) error {
    _, err := r.db.ExecContext(ctx, `UPDATE share_jobs SET status='running', updated_at=$1 WHERE id=$2 AND status='pending'`, time.Now().UTC(), jobID)
    return err
}

func (r *ShareRepository) MarkJobResult(ctx context.Context, jobID int64, success bool, errMsg *string) error {
    status := "failed"
    if success { status = "success" }
    _, err := r.db.ExecContext(ctx, `UPDATE share_jobs SET status=$1, attempts=attempts+1, last_error=$2, updated_at=$3 WHERE id=$4`, status, errMsg, time.Now().UTC(), jobID)
    return err
}

func (r *ShareRepository) UpdateRecordStatus(ctx context.Context, recordID int64, status string, errMsg *string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE video_share_records SET status=$1, error_message=$2, updated_at=$3 WHERE id=$4`, status, errMsg, time.Now().UTC(), recordID)
    return err
}

// Ensure interface compliance (compile-time)
var _ interface{ UpsertTrackShares(context.Context,string,string,[]string,string)([]*model.VideoShareRecord,error) } = (*ShareRepository)(nil)
