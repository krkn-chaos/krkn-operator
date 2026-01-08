from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class GetNodesRequest(_message.Message):
    __slots__ = ("kubeconfig_base64",)
    KUBECONFIG_BASE64_FIELD_NUMBER: _ClassVar[int]
    kubeconfig_base64: str
    def __init__(self, kubeconfig_base64: _Optional[str] = ...) -> None: ...

class GetNodesResponse(_message.Message):
    __slots__ = ("nodes",)
    NODES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, nodes: _Optional[_Iterable[str]] = ...) -> None: ...
