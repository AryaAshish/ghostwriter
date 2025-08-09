# Prompt Engine

AI-powered backend for generating personalized video scripts for creators (Instagram/YouTube) in India. Uses Go, Gin, GORM, SQLite, and OpenAI GPT for persona-based script generation.

---

## Overview

Prompt Engine enables:
- Onboarding creators with 20+ profile questions
- Persona and prompt generation (A/B/C variants)
- Saving all prompts and scripts to the database
- Generating final scripts using OpenAI GPT-4/3.5
- REST API with clean, modular Go code

---

## Installation & Running

1. **Clone the repo:**
   ```sh
   git clone <repo-url>
   cd ghostwriter
   ```
2. **Set up environment variables:**
   - Copy `.env.example` to `.env` and fill in your values (see below)
3. **Install dependencies:**
   ```sh
   go mod tidy
   ```
4. **Run the server:**
   ```sh
   go run ./cmd
   ```
   Server runs at `http://localhost:8080`

---

## Environment Variables

| Variable         | Description                        |
|------------------|------------------------------------|
| `DB_PATH`        | Path to SQLite DB file (e.g. `ghostwriter.db`) |
| `PORT`           | Port for HTTP server (default: 8080) |
| `OPENAI_API_KEY` | Your OpenAI API key                |
| `OPENAI_MODEL`   | OpenAI model (e.g. `gpt-4`, `gpt-3.5-turbo`) |

---

## API Documentation

### Health Check
- **GET /**
- **Response:** `{ "status": "ok" }`

---

### 1. Submit Creator Profile
- **POST /api/v1/submit-profile**
- **Body:**
```json
{
  "name": "Amit",
  "genre": "Comedy",
  "language": "Hindi",
  "email": "amit@example.com",
  "channel": "amitfunny",
  "region": "Delhi",
  "bio": "Standup comic",
  "goal": "Grow audience",
  "audience": "18-25",
  "tone": "Witty",
  "style": "Relatable",
  "inspiration": "Vir Das",
  "content_type": "Shorts",
  "frequency": "Weekly",
  "platform": "YouTube",
  "has_team": false,
  "experience": "2 years",
  "usp": "Quick punchlines",
  "other": ""
}
```
- **Response:** `{ "creator_id": 1, "message": "Profile submitted successfully" }`
- **Validation:** `name`, `genre`, `language` required

---

### 2. Generate Prompt
- **POST /api/v1/generate-prompt**
- **Body:**
```json
{
  "creator_id": 1,
  "topic": "How to go viral"
}
```
- **Response:**
```json
{
  "persona": "Amit is a Comedy creator from Delhi...",
  "prompt": "You are Amit... Write a script as if you are this creator, in their tone and style."
}
```
- **Validation:** `creator_id`, `topic` required

---

### 3. Generate A/B/C Prompt Variants
- **POST /api/v1/generate-prompt-ab**
- **Body:**
```json
{
  "creator_id": 1,
  "topic": "How to go viral"
}
```
- **Response:**
```json
{
  "persona": "Amit is a Comedy creator from Delhi...",
  "topic": "How to go viral",
  "variants": {
    "A": "You are Amit... Write a balanced, engaging script...",
    "B": "You are Amit... with extra punchlines and witty hooks...",
    "C": "You are Amit... focusing on storytelling and emotional depth..."
  }
}
```

---

### 4. Get Prompts by Creator
- **GET /api/v1/prompts/:creator_id**
- **Response:**
```json
[
  {
    "id": 2,
    "creator_id": "1",
    "topic": "How to go viral",
    "variant": "A",
    "prompt_text": "...",
    "created_at": "2025-07-30T12:40:00Z"
  }
]
```

---

### 5. Generate Script
- **POST /api/v1/generate-script**
- **Body:**
```json
{
  "creator_id": "1",
  "topic": "How to go viral",
  "variant": "A"
}
```
- **Response:**
```json
{
  "prompt_id": 2,
  "prompt": "...",
  "script": "...",
  "source": "gpt-4",
  "created_at": "2025-07-30T12:41:00Z"
}
```
- **Validation:** `creator_id`, `topic` required

---

### 6. Get Scripts by Creator
- **GET /api/v1/scripts/:creator_id**
- **Response:**
```json
[
  {
    "script_text": "...",
    "variant": "A",
    "source": "gpt-4",
    "created_at": "2025-07-30T12:41:00Z"
  }
]
```

---

## Error Handling
- All endpoints return meaningful error messages and 400/500 codes on invalid input or server errors.
- OpenAI API calls include retry logic and clear error reporting.

---

## License
MIT
