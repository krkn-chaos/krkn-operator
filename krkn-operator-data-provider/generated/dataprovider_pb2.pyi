from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
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

class ExecuteKubectlRequest(_message.Message):
    __slots__ = ("kubeconfig_base64", "command", "subcommand", "args", "flags", "boolean_flags", "timeout_seconds")
    class FlagsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    KUBECONFIG_BASE64_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    SUBCOMMAND_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    FLAGS_FIELD_NUMBER: _ClassVar[int]
    BOOLEAN_FLAGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    kubeconfig_base64: str
    command: str
    subcommand: str
    args: _containers.RepeatedScalarFieldContainer[str]
    flags: _containers.ScalarMap[str, str]
    boolean_flags: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    def __init__(self, kubeconfig_base64: _Optional[str] = ..., command: _Optional[str] = ..., subcommand: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., flags: _Optional[_Mapping[str, str]] = ..., boolean_flags: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class ExecuteKubectlResponse(_message.Message):
    __slots__ = ("stdout_base64", "stderr_base64", "exit_code", "error")
    STDOUT_BASE64_FIELD_NUMBER: _ClassVar[int]
    STDERR_BASE64_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    stdout_base64: str
    stderr_base64: str
    exit_code: int
    error: str
    def __init__(self, stdout_base64: _Optional[str] = ..., stderr_base64: _Optional[str] = ..., exit_code: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...
