
```http request
curl -X POST   http://localhost:8000/flag   -H 'Content-Type: application/json'   -d '{
"flag_name": "feature_new_ui",
"is_enable": true,
"active_from": "2025-04-05T00:00:00Z",
"data": {"color": "blue", "size": "large"},
"default_data": {"color": "gray", "size": "medium"},
"created_user": "123e4567-e89b-12d3-a456-426614174000",
"created_at": "2025-04-01T10:00:00Z",
"updated_at": "2025-04-01T10:00:00Z"
}'

curl -X PUT  http://localhost:8000/flag/feature_new_ui   -H 'Content-Type: application/json'   -d '{
"flag_name": "feature_new_ui",
"is_enable": false,
"active_from": "2025-04-05T00:00:00Z",
"data": {"color": "blue", "size": "large"},
"default_data": {"color": "gray1", "size": "medium1"},
"created_user": "123e4567-e89b-12d3-a456-426614174000",
"created_at": "2025-04-01T10:00:00Z",
"updated_at": "2025-04-01T10:00:00Z"
}'

curl http://localhost:8000/flag/feature_new_ui 

curl -X DELETE http://localhost:8000/flag/feature_new_ui 


# unknown flags
curl -X POST \
  http://localhost:8000/flags \
  -H 'Content-Type: application/json' \
  -d '{
    "flag_names": [
      "feature_new_ui",
      "dark_mode",
      "beta_access",
      "promo_banner_2025"
    ]
  }'

# exist flags
curl -X POST \
  http://localhost:8000/flags \
  -H 'Content-Type: application/json' \
  -d '{
    "flag_names": [
      "2feature_new_ui"      
    ]
  }'
  
  curl -X POST \
  http://localhost:8000/flags \
  -H 'Content-Type: application/json' \
  -d '{
    "flag_names": [
      "2feature_new_ui",
      "feature_new_ui1"      
    ]
  }'
```