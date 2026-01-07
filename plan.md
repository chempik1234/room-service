1. simple protobuf
2. in-memory room implementation
    1. ports, repos
    2. rooms exist offline for TTL
3. handlers (stream for room lifecycle)
4. USER_ONLY_1_ROOM=bool
5. Query (e.g. list all rooms, get rooms meta)
6. Tell if error caused by caller or is it internal
7. Nested keys like data_item1,0,key1,1 is for data["data_item1"][0]["key1"][1]
8. more...