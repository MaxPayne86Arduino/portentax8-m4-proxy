## git clone github.com/msgpack-rpc/msgpack-rpc-python
## sudo python setup.py install

import msgpackrpc

client = msgpackrpc.Client(msgpackrpc.Address("localhost", 5000))
result = client.call('register', 5005, ['multiply', 'divide'])

class Server(object):
    def multiply(self, x, y):
        return x * y
    def divide(self, x, y):
        return x / y

server = msgpackrpc.Server(Server())
server.listen(msgpackrpc.Address("localhost", 5005))
server.start()
