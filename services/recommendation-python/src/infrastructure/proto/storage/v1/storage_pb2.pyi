from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PutObjectRequest(_message.Message):
    __slots__ = ("metadata", "data")
    METADATA_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    metadata: PutObjectMetadata
    data: bytes
    def __init__(self, metadata: _Optional[_Union[PutObjectMetadata, _Mapping]] = ..., data: _Optional[bytes] = ...) -> None: ...

class PutObjectMetadata(_message.Message):
    __slots__ = ("reference", "content_type")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    reference: str
    content_type: str
    def __init__(self, reference: _Optional[str] = ..., content_type: _Optional[str] = ...) -> None: ...

class PutObjectResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPrivateObjectRequest(_message.Message):
    __slots__ = ("reference",)
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    reference: str
    def __init__(self, reference: _Optional[str] = ...) -> None: ...

class GetPrivateObjectResponse(_message.Message):
    __slots__ = ("content_type", "data")
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    content_type: str
    data: bytes
    def __init__(self, content_type: _Optional[str] = ..., data: _Optional[bytes] = ...) -> None: ...

class DeleteObjectRequest(_message.Message):
    __slots__ = ("reference",)
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    reference: str
    def __init__(self, reference: _Optional[str] = ...) -> None: ...

class DeleteObjectResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HeadObjectRequest(_message.Message):
    __slots__ = ("reference",)
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    reference: str
    def __init__(self, reference: _Optional[str] = ...) -> None: ...

class HeadObjectResponse(_message.Message):
    __slots__ = ("content_type",)
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    content_type: str
    def __init__(self, content_type: _Optional[str] = ...) -> None: ...

class GetModelRequest(_message.Message):
    __slots__ = ("model_id", "revision", "embedding_dimension")
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_DIMENSION_FIELD_NUMBER: _ClassVar[int]
    model_id: str
    revision: str
    embedding_dimension: int
    def __init__(self, model_id: _Optional[str] = ..., revision: _Optional[str] = ..., embedding_dimension: _Optional[int] = ...) -> None: ...

class GetModelResponse(_message.Message):
    __slots__ = ("data", "content_type")
    DATA_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    content_type: str
    def __init__(self, data: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...
