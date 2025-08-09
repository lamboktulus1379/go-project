package persistence

import (
	"context"
	"database/sql"
	"my-project/domain/model"
	"time"
)

type OAuthTokenRepository struct{ db *sql.DB }

func NewOAuthTokenRepository(db *sql.DB) *OAuthTokenRepository { return &OAuthTokenRepository{db: db} }

func (r *OAuthTokenRepository) UpsertToken(ctx context.Context, t *model.OAuthToken) error {
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	q := `INSERT INTO oauth_tokens (user_id, platform, access_token, refresh_token, expires_at, scopes, page_id, page_name, token_type, created_at, updated_at)
		  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		  ON CONFLICT (user_id, platform) DO UPDATE SET
			access_token=EXCLUDED.access_token,
			refresh_token=EXCLUDED.refresh_token,
			expires_at=EXCLUDED.expires_at,
			scopes=EXCLUDED.scopes,
			page_id=EXCLUDED.page_id,
			page_name=EXCLUDED.page_name,
			token_type=EXCLUDED.token_type,
			updated_at=EXCLUDED.updated_at` 
	_, err := r.db.ExecContext(ctx, q, t.UserID, t.Platform, t.AccessToken, t.RefreshToken, t.ExpiresAt, t.Scopes, t.PageID, t.PageName, t.TokenType, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *OAuthTokenRepository) GetToken(ctx context.Context, userID, platform string) (*model.OAuthToken, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, platform, access_token, refresh_token, expires_at, scopes, page_id, page_name, token_type, created_at, updated_at FROM oauth_tokens WHERE user_id=$1 AND platform=$2`, userID, platform)
	tok := &model.OAuthToken{}
	var exp sql.NullTime
	var pageID, pageName, tokenType sql.NullString
	if err := row.Scan(&tok.ID, &tok.UserID, &tok.Platform, &tok.AccessToken, &tok.RefreshToken, &exp, &tok.Scopes, &pageID, &pageName, &tokenType, &tok.CreatedAt, &tok.UpdatedAt); err != nil {
		return nil, err
	}
	if exp.Valid {
		tok.ExpiresAt = &exp.Time
	}
	if pageID.Valid { v := pageID.String; tok.PageID = &v }
	if pageName.Valid { v := pageName.String; tok.PageName = &v }
	if tokenType.Valid { v := tokenType.String; tok.TokenType = &v }
	return tok, nil
}
