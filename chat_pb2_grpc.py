# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
"""Client and server classes corresponding to protobuf-defined services."""
import grpc
import warnings

import chat_pb2 as chat__pb2

GRPC_GENERATED_VERSION = '1.71.0'
GRPC_VERSION = grpc.__version__
_version_not_supported = False

try:
    from grpc._utilities import first_version_is_lower
    _version_not_supported = first_version_is_lower(GRPC_VERSION, GRPC_GENERATED_VERSION)
except ImportError:
    _version_not_supported = True

if _version_not_supported:
    raise RuntimeError(
        f'The grpc package installed is at version {GRPC_VERSION},'
        + f' but the generated code in chat_pb2_grpc.py depends on'
        + f' grpcio>={GRPC_GENERATED_VERSION}.'
        + f' Please upgrade your grpc module to grpcio>={GRPC_GENERATED_VERSION}'
        + f' or downgrade your generated code using grpcio-tools<={GRPC_VERSION}.'
    )


class ChatServiceStub(object):
    """Definisi layanan gRPC
    """

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.Login = channel.unary_unary(
                '/chat.ChatService/Login',
                request_serializer=chat__pb2.LoginRequest.SerializeToString,
                response_deserializer=chat__pb2.LoginResponse.FromString,
                _registered_method=True)
        self.GetRecentMessages = channel.unary_stream(
                '/chat.ChatService/GetRecentMessages',
                request_serializer=chat__pb2.Empty.SerializeToString,
                response_deserializer=chat__pb2.ChatMessage.FromString,
                _registered_method=True)
        self.SendMultipleMessages = channel.stream_unary(
                '/chat.ChatService/SendMultipleMessages',
                request_serializer=chat__pb2.MessageRequest.SerializeToString,
                response_deserializer=chat__pb2.MessageResponse.FromString,
                _registered_method=True)
        self.ChatStream = channel.stream_stream(
                '/chat.ChatService/ChatStream',
                request_serializer=chat__pb2.MessageRequest.SerializeToString,
                response_deserializer=chat__pb2.ChatMessage.FromString,
                _registered_method=True)


class ChatServiceServicer(object):
    """Definisi layanan gRPC
    """

    def Login(self, request, context):
        """Unary RPC - User mendaftar/melakukan login
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def GetRecentMessages(self, request, context):
        """Server Streaming RPC - Saat user masuk chat room, server mengirim pesan terbaru
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def SendMultipleMessages(self, request_iterator, context):
        """Client Streaming RPC - User bisa mengirim beberapa pesan sebelum mendapat respons
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def ChatStream(self, request_iterator, context):
        """Bidirectional Streaming RPC - Chat real-time antara banyak user
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')


def add_ChatServiceServicer_to_server(servicer, server):
    rpc_method_handlers = {
            'Login': grpc.unary_unary_rpc_method_handler(
                    servicer.Login,
                    request_deserializer=chat__pb2.LoginRequest.FromString,
                    response_serializer=chat__pb2.LoginResponse.SerializeToString,
            ),
            'GetRecentMessages': grpc.unary_stream_rpc_method_handler(
                    servicer.GetRecentMessages,
                    request_deserializer=chat__pb2.Empty.FromString,
                    response_serializer=chat__pb2.ChatMessage.SerializeToString,
            ),
            'SendMultipleMessages': grpc.stream_unary_rpc_method_handler(
                    servicer.SendMultipleMessages,
                    request_deserializer=chat__pb2.MessageRequest.FromString,
                    response_serializer=chat__pb2.MessageResponse.SerializeToString,
            ),
            'ChatStream': grpc.stream_stream_rpc_method_handler(
                    servicer.ChatStream,
                    request_deserializer=chat__pb2.MessageRequest.FromString,
                    response_serializer=chat__pb2.ChatMessage.SerializeToString,
            ),
    }
    generic_handler = grpc.method_handlers_generic_handler(
            'chat.ChatService', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))
    server.add_registered_method_handlers('chat.ChatService', rpc_method_handlers)


 # This class is part of an EXPERIMENTAL API.
class ChatService(object):
    """Definisi layanan gRPC
    """

    @staticmethod
    def Login(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_unary(
            request,
            target,
            '/chat.ChatService/Login',
            chat__pb2.LoginRequest.SerializeToString,
            chat__pb2.LoginResponse.FromString,
            options,
            channel_credentials,
            insecure,
            call_credentials,
            compression,
            wait_for_ready,
            timeout,
            metadata,
            _registered_method=True)

    @staticmethod
    def GetRecentMessages(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_stream(
            request,
            target,
            '/chat.ChatService/GetRecentMessages',
            chat__pb2.Empty.SerializeToString,
            chat__pb2.ChatMessage.FromString,
            options,
            channel_credentials,
            insecure,
            call_credentials,
            compression,
            wait_for_ready,
            timeout,
            metadata,
            _registered_method=True)

    @staticmethod
    def SendMultipleMessages(request_iterator,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.stream_unary(
            request_iterator,
            target,
            '/chat.ChatService/SendMultipleMessages',
            chat__pb2.MessageRequest.SerializeToString,
            chat__pb2.MessageResponse.FromString,
            options,
            channel_credentials,
            insecure,
            call_credentials,
            compression,
            wait_for_ready,
            timeout,
            metadata,
            _registered_method=True)

    @staticmethod
    def ChatStream(request_iterator,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.stream_stream(
            request_iterator,
            target,
            '/chat.ChatService/ChatStream',
            chat__pb2.MessageRequest.SerializeToString,
            chat__pb2.ChatMessage.FromString,
            options,
            channel_credentials,
            insecure,
            call_credentials,
            compression,
            wait_for_ready,
            timeout,
            metadata,
            _registered_method=True)
