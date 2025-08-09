package persistence

import (
    "context"
    "database/sql"
    "time"
    "my-project/domain/model"
)

type OAuthTokenRepository struct { db *sql.DB }

func NewOAuthTokenRepository(db *sql.DB) *OAuthTokenRepository { return &OAuthTokenRepository{db: db} }

func (r *OAuthTokenRepository) UpsertToken(ctx context.Context, t *model.OAuthToken) error {
    now := time.Now().UTC()
    if t.CreatedAt.IsZero() { t.CreatedAt = now }
    t.UpdatedAt = now
    q := `INSERT INTO oauth_tokens (user_id, platform, access_token, refresh_token, expires_at, scopes, created_at, updated_at)
          VALUES (?,?,?,?,?,?,?,?)
          ON DUPLICATE KEY UPDATE access_token=VALUES(access_token), refresh_token=VALUES(refresh_token), expires_at=VALUES(expires_at), scopes=VALUES(scopes), updated_at=VALUES(updated_at)`
    _, err := r.db.ExecContext(ctx, q, t.UserID, t.Platform, t.AccessToken, t.RefreshToken, t.ExpiresAt, t.Scopes, t.CreatedAt, t.UpdatedAt)
    return err
}

func (r *OAuthTokenRepository) GetToken(ctx context.Context, userID, platform string) (*model.OAuthToken, error) {
    row := r.db.QueryRowContext(ctx, `SELECT id, user_id, platform, access_token, refresh_token, expires_at, scopes, created_at, updated_at FROM oauth_tokens WHERE user_id=? AND platform=?`, userID, platform)
    tok := &model.OAuthToken{}
    var exp sql.NullTime
    if err := row.Scan(&tok.ID, &tok.UserID, &tok.Platform, &tok.AccessToken, &tok.RefreshToken, &exp, &tok.Scopes, &tok.CreatedAt, &tok.UpdatedAt); err != nil {
        return nil, err
    }
    if exp.Valid { tok.ExpiresAt = &exp.Time }
    return tok, nil
}
