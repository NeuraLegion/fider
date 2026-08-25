# 🌟 Bright Star — Run Memory

<!-- BRIGHT_STAR_DATA — generated; do not edit -->
```json
{
  "version": 1,
  "generatedAt": "2026-08-25T19:23:56.671Z",
  "techStack": {
    "languages": [
      "JavaScript",
      "TypeScript",
      "Go"
    ],
    "frameworks": [],
    "databases": [
      "PostgreSQL"
    ]
  },
  "startup": {
    "command": "docker compose -f compose.yml up -d --build",
    "port": 3000,
    "prerequisites": [],
    "envVars": {},
    "healthCheckPath": "/signup"
  },
  "setup": {
    "completed": false
  },
  "auth": {
    "hasAuth": true,
    "authObjectId": "qdCQfzjjgAYxXTtGGqnAgi"
  },
  "hints": {
    "startup": [
      "Fider starts reproducibly with docker compose -f compose.yml up -d (after docker compose -f compose.yml up -d --build). App is published at localhost:3000; PostgreSQL is the compose postgres service at postgres:5432. Required local env is already in compose.yml, including DATABASE_URL, JWT_SECRET, BASE_URL, PORT=3000, HOST=0.0.0.0. Root redirects 307 to /signup; app logs confirm HTTP server on 0.0.0.0:3000 and migrations applied."
    ]
  }
}
```
<!-- BRIGHT_STAR_DATA -->
