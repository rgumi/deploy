# TODO

[x] 1. finish switchover
[x] 2. add better methods for metric retrievel

3. test other strategies

[x] 4. add handlers for API
[x] 5. better workflow for creating routes/backends
[x] 6. dynamic reloading of routes when backend is added
[x] 7. add increasing counter for prom/dashboard

# Observed Errors

...

# Fixed Errors

[x] 1. Storage Lock
[x] 2. Weird Weight update to [105, 251] when using switchover?! => uint8 overflow due to race condition in backend creation
