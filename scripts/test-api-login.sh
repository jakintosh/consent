#!/usr/bin/bash
curl -i -X POST \
	-H "Content-Type: application/json" \
	-d '{"username":"jakintosh","password":"password"}' \
	http://localhost:9001/api/login
