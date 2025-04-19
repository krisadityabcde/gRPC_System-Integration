// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.30.1
// source: proto/chat.proto

package __

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ActiveUsersUpdate_UpdateType int32

const (
	ActiveUsersUpdate_FULL_LIST ActiveUsersUpdate_UpdateType = 0 // Full list of active users
	ActiveUsersUpdate_JOIN      ActiveUsersUpdate_UpdateType = 1 // User joined
	ActiveUsersUpdate_LEAVE     ActiveUsersUpdate_UpdateType = 2 // User left
)

// Enum value maps for ActiveUsersUpdate_UpdateType.
var (
	ActiveUsersUpdate_UpdateType_name = map[int32]string{
		0: "FULL_LIST",
		1: "JOIN",
		2: "LEAVE",
	}
	ActiveUsersUpdate_UpdateType_value = map[string]int32{
		"FULL_LIST": 0,
		"JOIN":      1,
		"LEAVE":     2,
	}
)

func (x ActiveUsersUpdate_UpdateType) Enum() *ActiveUsersUpdate_UpdateType {
	p := new(ActiveUsersUpdate_UpdateType)
	*p = x
	return p
}

func (x ActiveUsersUpdate_UpdateType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ActiveUsersUpdate_UpdateType) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_chat_proto_enumTypes[0].Descriptor()
}

func (ActiveUsersUpdate_UpdateType) Type() protoreflect.EnumType {
	return &file_proto_chat_proto_enumTypes[0]
}

func (x ActiveUsersUpdate_UpdateType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ActiveUsersUpdate_UpdateType.Descriptor instead.
func (ActiveUsersUpdate_UpdateType) EnumDescriptor() ([]byte, []int) {
	return file_proto_chat_proto_rawDescGZIP(), []int{4, 0}
}

type LoginRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Username      string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoginRequest) Reset() {
	*x = LoginRequest{}
	mi := &file_proto_chat_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoginRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginRequest) ProtoMessage() {}

func (x *LoginRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_chat_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginRequest.ProtoReflect.Descriptor instead.
func (*LoginRequest) Descriptor() ([]byte, []int) {
	return file_proto_chat_proto_rawDescGZIP(), []int{0}
}

func (x *LoginRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

type LoginResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Username      string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Message       string                 `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoginResponse) Reset() {
	*x = LoginResponse{}
	mi := &file_proto_chat_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoginResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginResponse) ProtoMessage() {}

func (x *LoginResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_chat_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginResponse.ProtoReflect.Descriptor instead.
func (*LoginResponse) Descriptor() ([]byte, []int) {
	return file_proto_chat_proto_rawDescGZIP(), []int{1}
}

func (x *LoginResponse) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *LoginResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type ChatMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Sender        string                 `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	Message       string                 `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	Timestamp     string                 `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChatMessage) Reset() {
	*x = ChatMessage{}
	mi := &file_proto_chat_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChatMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChatMessage) ProtoMessage() {}

func (x *ChatMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_chat_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChatMessage.ProtoReflect.Descriptor instead.
func (*ChatMessage) Descriptor() ([]byte, []int) {
	return file_proto_chat_proto_rawDescGZIP(), []int{2}
}

func (x *ChatMessage) GetSender() string {
	if x != nil {
		return x.Sender
	}
	return ""
}

func (x *ChatMessage) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *ChatMessage) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

// New message types for active users streaming
type ActiveUsersRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Username      string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"` // The requesting user's name
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ActiveUsersRequest) Reset() {
	*x = ActiveUsersRequest{}
	mi := &file_proto_chat_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ActiveUsersRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActiveUsersRequest) ProtoMessage() {}

func (x *ActiveUsersRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_chat_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActiveUsersRequest.ProtoReflect.Descriptor instead.
func (*ActiveUsersRequest) Descriptor() ([]byte, []int) {
	return file_proto_chat_proto_rawDescGZIP(), []int{3}
}

func (x *ActiveUsersRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

type ActiveUsersUpdate struct {
	state         protoimpl.MessageState       `protogen:"open.v1"`
	UpdateType    ActiveUsersUpdate_UpdateType `protobuf:"varint,1,opt,name=update_type,json=updateType,proto3,enum=grpcchat.ActiveUsersUpdate_UpdateType" json:"update_type,omitempty"`
	Users         []string                     `protobuf:"bytes,2,rep,name=users,proto3" json:"users,omitempty"`       // For FULL_LIST, the complete list of users
	Username      string                       `protobuf:"bytes,3,opt,name=username,proto3" json:"username,omitempty"` // For JOIN/LEAVE, the affected user
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ActiveUsersUpdate) Reset() {
	*x = ActiveUsersUpdate{}
	mi := &file_proto_chat_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ActiveUsersUpdate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActiveUsersUpdate) ProtoMessage() {}

func (x *ActiveUsersUpdate) ProtoReflect() protoreflect.Message {
	mi := &file_proto_chat_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActiveUsersUpdate.ProtoReflect.Descriptor instead.
func (*ActiveUsersUpdate) Descriptor() ([]byte, []int) {
	return file_proto_chat_proto_rawDescGZIP(), []int{4}
}

func (x *ActiveUsersUpdate) GetUpdateType() ActiveUsersUpdate_UpdateType {
	if x != nil {
		return x.UpdateType
	}
	return ActiveUsersUpdate_FULL_LIST
}

func (x *ActiveUsersUpdate) GetUsers() []string {
	if x != nil {
		return x.Users
	}
	return nil
}

func (x *ActiveUsersUpdate) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

var File_proto_chat_proto protoreflect.FileDescriptor

const file_proto_chat_proto_rawDesc = "" +
	"\n" +
	"\x10proto/chat.proto\x12\bgrpcchat\"*\n" +
	"\fLoginRequest\x12\x1a\n" +
	"\busername\x18\x01 \x01(\tR\busername\"E\n" +
	"\rLoginResponse\x12\x1a\n" +
	"\busername\x18\x01 \x01(\tR\busername\x12\x18\n" +
	"\amessage\x18\x02 \x01(\tR\amessage\"]\n" +
	"\vChatMessage\x12\x16\n" +
	"\x06sender\x18\x01 \x01(\tR\x06sender\x12\x18\n" +
	"\amessage\x18\x02 \x01(\tR\amessage\x12\x1c\n" +
	"\ttimestamp\x18\x03 \x01(\tR\ttimestamp\"0\n" +
	"\x12ActiveUsersRequest\x12\x1a\n" +
	"\busername\x18\x01 \x01(\tR\busername\"\xc0\x01\n" +
	"\x11ActiveUsersUpdate\x12G\n" +
	"\vupdate_type\x18\x01 \x01(\x0e2&.grpcchat.ActiveUsersUpdate.UpdateTypeR\n" +
	"updateType\x12\x14\n" +
	"\x05users\x18\x02 \x03(\tR\x05users\x12\x1a\n" +
	"\busername\x18\x03 \x01(\tR\busername\"0\n" +
	"\n" +
	"UpdateType\x12\r\n" +
	"\tFULL_LIST\x10\x00\x12\b\n" +
	"\x04JOIN\x10\x01\x12\t\n" +
	"\x05LEAVE\x10\x022\xd9\x01\n" +
	"\vChatService\x128\n" +
	"\x05Login\x12\x16.grpcchat.LoginRequest\x1a\x17.grpcchat.LoginResponse\x12>\n" +
	"\n" +
	"ChatStream\x12\x15.grpcchat.ChatMessage\x1a\x15.grpcchat.ChatMessage(\x010\x01\x12P\n" +
	"\x11ActiveUsersStream\x12\x1c.grpcchat.ActiveUsersRequest\x1a\x1b.grpcchat.ActiveUsersUpdate0\x01B\x03Z\x01.b\x06proto3"

var (
	file_proto_chat_proto_rawDescOnce sync.Once
	file_proto_chat_proto_rawDescData []byte
)

func file_proto_chat_proto_rawDescGZIP() []byte {
	file_proto_chat_proto_rawDescOnce.Do(func() {
		file_proto_chat_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_chat_proto_rawDesc), len(file_proto_chat_proto_rawDesc)))
	})
	return file_proto_chat_proto_rawDescData
}

var file_proto_chat_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_chat_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_proto_chat_proto_goTypes = []any{
	(ActiveUsersUpdate_UpdateType)(0), // 0: grpcchat.ActiveUsersUpdate.UpdateType
	(*LoginRequest)(nil),              // 1: grpcchat.LoginRequest
	(*LoginResponse)(nil),             // 2: grpcchat.LoginResponse
	(*ChatMessage)(nil),               // 3: grpcchat.ChatMessage
	(*ActiveUsersRequest)(nil),        // 4: grpcchat.ActiveUsersRequest
	(*ActiveUsersUpdate)(nil),         // 5: grpcchat.ActiveUsersUpdate
}
var file_proto_chat_proto_depIdxs = []int32{
	0, // 0: grpcchat.ActiveUsersUpdate.update_type:type_name -> grpcchat.ActiveUsersUpdate.UpdateType
	1, // 1: grpcchat.ChatService.Login:input_type -> grpcchat.LoginRequest
	3, // 2: grpcchat.ChatService.ChatStream:input_type -> grpcchat.ChatMessage
	4, // 3: grpcchat.ChatService.ActiveUsersStream:input_type -> grpcchat.ActiveUsersRequest
	2, // 4: grpcchat.ChatService.Login:output_type -> grpcchat.LoginResponse
	3, // 5: grpcchat.ChatService.ChatStream:output_type -> grpcchat.ChatMessage
	5, // 6: grpcchat.ChatService.ActiveUsersStream:output_type -> grpcchat.ActiveUsersUpdate
	4, // [4:7] is the sub-list for method output_type
	1, // [1:4] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proto_chat_proto_init() }
func file_proto_chat_proto_init() {
	if File_proto_chat_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_chat_proto_rawDesc), len(file_proto_chat_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_chat_proto_goTypes,
		DependencyIndexes: file_proto_chat_proto_depIdxs,
		EnumInfos:         file_proto_chat_proto_enumTypes,
		MessageInfos:      file_proto_chat_proto_msgTypes,
	}.Build()
	File_proto_chat_proto = out.File
	file_proto_chat_proto_goTypes = nil
	file_proto_chat_proto_depIdxs = nil
}
