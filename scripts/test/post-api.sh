#!/usr/bin/bash

curl -X POST http://localhost:9001/api/login
curl -X POST http://localhost:9001/api/logout
curl -X POST http://localhost:9001/api/refresh
curl -X POST http://localhost:9001/api/register
