```
openssl genrsa -out private_key 2048
openssl req -new -x509 -key private_key -out public_key -days 365
```
