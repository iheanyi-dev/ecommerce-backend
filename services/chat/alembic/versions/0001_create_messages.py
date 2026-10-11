"""Create the messages table."""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

revision = "0001_create_messages"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create the messages table and conversation lookup index."""
    op.create_table(
        "messages",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            nullable=False,
        ),
        sa.Column(
            "conversation_id",
            postgresql.UUID(as_uuid=True),
            nullable=False,
        ),
        sa.Column(
            "store_id",
            postgresql.UUID(as_uuid=True),
            nullable=False,
        ),
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            nullable=True,
        ),
        sa.Column("role", sa.String(length=32), nullable=False),
        sa.Column("content", sa.Text(), nullable=False),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
        ),
    )

    op.create_index(
        "ix_messages_store_conversation_created",
        "messages",
        ["store_id", "conversation_id", "created_at", "id"],
        unique=False,
    )


def downgrade() -> None:
    """Remove the messages table."""
    op.drop_index(
        "ix_messages_store_conversation_created",
        table_name="messages",
    )
    op.drop_table("messages")
