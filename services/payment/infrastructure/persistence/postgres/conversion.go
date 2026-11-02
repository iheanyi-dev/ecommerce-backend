package postgres

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func parseUUID(value string) (pgtype.UUID, error) {
	var result pgtype.UUID
	if err := result.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse UUID %q: %w", value, err)
	}
	return result, nil
}

func nullableUUID(value string) (pgtype.UUID, error) {
	if value == "" {
		return pgtype.UUID{}, nil
	}
	return parseUUID(value)
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}

	raw := hex.EncodeToString(value.Bytes[:])
	return raw[0:8] + "-" + raw[8:12] + "-" + raw[12:16] +
		"-" + raw[16:20] + "-" + raw[20:32]
}

func nullableText(value string) pgtype.Text {
	return pgtype.Text{
		String: value,
		Valid:  value != "",
	}
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
