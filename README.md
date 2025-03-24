# gRPC - Chat Room
## Instalasi:
```python
python -m venv venv
venv\Scripts\activate
pip install -r requirements.txt
```

## Langkah Kerja:
1. Awalnya semua jenis message didefinisikan di `chat.proto`
2. Semua jenis service yang dibutuhkan juga didefinisikan di `chat.proto`
3. Dengan `python -m grpc_tools.protoc -I=proto --python_out=. --grpc_python_out=. proto/chat.proto` berfungsi untuk generate kode gRPC dari protobuff yang dibuat. `chat_pb2.py` berisi definisi message yang sebelumnya dibuat. Sementara `chat_pb2_grpc.py` berisi metode/service (stub untuk client bisa menghubungi server)
4. Membuat kode untuk Server dan Client, lalu menjalankan client setelah server berjalan