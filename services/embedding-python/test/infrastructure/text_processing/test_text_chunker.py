"""Unit tests for the deterministic character text chunker."""

import pytest

from infrastructure.text_processing import CharacterTextChunker


def test_short_text_returns_one_unchanged_chunk():
    chunker = CharacterTextChunker(chunk_size=10, overlap=2)

    assert chunker.chunk("hello") == ("hello",)


def test_long_text_is_split_into_ordered_overlapping_chunks():
    chunker = CharacterTextChunker(chunk_size=6, overlap=2)

    chunks = chunker.chunk("abcdefghijkl")

    assert chunks == ("abcdef", "efghij", "ijkl")


def test_overlap_preserves_characters_between_adjacent_chunks():
    chunker = CharacterTextChunker(chunk_size=5, overlap=2)

    chunks = chunker.chunk("abcdefgh")

    assert chunks == ("abcde", "defgh")
    assert chunks[0][-2:] == chunks[1][:2]


def test_chunking_preserves_unicode_characters():
    chunker = CharacterTextChunker(chunk_size=3, overlap=1)

    assert chunker.chunk("猫犬鳥魚") == ("猫犬鳥", "鳥魚")


def test_empty_and_whitespace_only_text_produce_no_chunks():
    chunker = CharacterTextChunker(chunk_size=10, overlap=2)

    assert chunker.chunk("") == ()
    assert chunker.chunk("   \n\t ") == ()


def test_whitespace_inside_document_is_preserved():
    chunker = CharacterTextChunker(chunk_size=4, overlap=1)

    assert chunker.chunk("ab cd ef") == ("ab c", "cd e", "ef")


@pytest.mark.parametrize(
    ("chunk_size", "overlap"),
    [
        (0, 0),
        (-1, 0),
        (5, -1),
        (5, 5),
        (5, 6),
    ],
)
def test_invalid_chunking_configuration_is_rejected(chunk_size, overlap):
    with pytest.raises(ValueError):
        CharacterTextChunker(chunk_size=chunk_size, overlap=overlap)


@pytest.mark.parametrize(
    ("chunk_size", "overlap"),
    [
        (True, 0),
        (10, False),
        (2.5, 0),
        (10, 1.5),
    ],
)
def test_non_integer_configuration_is_rejected(chunk_size, overlap):
    with pytest.raises(TypeError):
        CharacterTextChunker(chunk_size=chunk_size, overlap=overlap)


def test_non_string_input_is_rejected():
    chunker = CharacterTextChunker()

    with pytest.raises(TypeError, match="text must be a string"):
        chunker.chunk(None)
