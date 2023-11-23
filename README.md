# hraft

hraft a example use of the [hashicorp/raft](https://github.com/hashicorp/raft).  
Referring to https://github.com/otoolep/hraftd for my own learning.

# Build

```
make build
```

# Running cluster

```
$ make run/node1
$ make run/node2
$ make run/node3
```

```
$ curl http://127.0.0.1:21001/kv/hello -X PUT --data "world"
$ curl http://127.0.0.1:21001/kv/hello
{"hello":"world"}
```

If node1 goes down, the cluster can be maintained. However, if two or more nodes go down, writing will not be possible.
