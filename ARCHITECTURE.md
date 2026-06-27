# Ghostwriter Prompt Engine — Architecture & File Documentation

## Project Overview

**Ghostwriter Prompt Engine** is a Go REST API backend that powers AI-driven, personalized video script generation for Instagram/YouTube creators in India. Creators onboard with a detailed profile, and the system uses that profile to build a "persona," generate tailored prompts (with A/B/C variant testing), and then call OpenAI to produce ready-to-use video scripts.

### Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.16 |
| HTTP Framework | Gin |
| ORM | GORM |
| Database | SQLite |
| AI Provider | OpenAI (GPT-4 / GPT-3.5) |
| API Docs | Swagger (swaggo) |

### High-Level Architecture

```
┌─────────────┐     ┌──────────┐     ┌──────────────┐     ┌─────────┐
│   Client    │────▶│  Router  │────▶│   Handlers   │────▶│Services │
│ (REST API)  │◀────│  (Gin)   │◀────│  (HTTP I/O)  │◀────│(Logic)  │
└─────────────┘     └──────────┘     └──────────────┘     └────┬────┘
                                                                │
                                              ┌─────────────────┼──────────────┐
                                              ▼                 ▼              ▼
                                        ┌──────────┐    ┌────────────┐  ┌──────────┐
                                        │  SQLite  │    │  OpenAI    │  │  Models  │
                                        │  (GORM)  │    │  API       │  │  (DTOs)  │
                                        └──────────┘    └────────────┘  └──────────┘
```

---

## API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/` | Health check |
| GET | `/api/v1/onboarding/questions` | Style questionnaire (optional `voice_input_path`, `genre`) |
| POST | `/api/v1/submit-profile` | Onboard profile + persona (three voice paths) |
| GET | `/api/v1/persona/:creator_id` | Get persona with fingerprint metadata |
| PATCH | `/api/v1/persona/:creator_id` | Manual persona edits |
| POST | `/api/v1/persona/:creator_id/analyze-voice` | Re-run voice analysis on samples |
| POST | `/api/v1/generate-prompt` | Generate a single base prompt for a creator + topic |
| POST | `/api/v1/generate-prompt-ab` | Generate A/B/C variant prompts for testing |
| GET | `/api/v1/prompts/:creator_id` | Retrieve all prompts for a creator |
| POST | `/api/v1/generate-script` | Generate a video script via OpenAI from a stored prompt |
| GET | `/api/v1/scripts/:creator_id` | Retrieve all scripts for a creator |
| POST | `/api/v1/scripts/:script_id/feedback` | Feedback loop + calibration sample ingestion |
| GET | `/app/` | Persona Lab UI (manual ChatGPT testing) |

---

## Voice fingerprint pipeline (three paths)

Creators choose one of three onboarding paths via `voice_input_path`:

| Path | Input | Initial `voice_mode` | Prompt strategy |
|------|-------|----------------------|-----------------|
| `paste_scripts` | 2+ samples or 300+ words | `derived` | Local stylometry + full few-shot examples in system prompt |
| `guided_write` | 3 guided exercises (40+ words each) | `calibrated` / `derived` | Same once word threshold met |
| `skip_calibrate` | Questionnaire + preferred/avoid + anti-voice | `declared` | Score bands + niche rules; calibrate via feedback |

**Local extraction** (`services/voice_extractor.go`): function-word frequencies, sentence length, filler density, Hinglish ratio, hook pattern — no OpenAI required.

**Calibration loop**: `POST /scripts/:id/feedback` accepts `edited_script`, `generated_script`, and structured `toggles`. Edited scripts become `calibration_edit` samples; persona mode and confidence promote over time.

**Coverage gate**: `scripts/check_coverage.sh` enforces ≥95% line + function coverage on `handlers`, `services`, `router`, `db`, `utils`.

---

### Entry Point

| File | Intent |
|------|--------|
| `cmd/main.go` | Application bootstrap. Loads `.env`, initializes SQLite, performs dependency injection (wires services → handlers), sets up CORS middleware, registers Swagger route, mounts all API routes, and starts the Gin server on the configured port. |

### Database Layer

| File | Intent |
|------|--------|
| `db/connection.go` | Opens the SQLite database using GORM and runs `AutoMigrate` for all models (`CreatorProfile`, `Prompt`, `Script`). Falls back to `ghostwriter.db` if `DB_PATH` env is unset. |

### Router

| File | Intent |
|------|--------|
| `router/routes.go` | Central route registration. Groups all business endpoints under `/api/v1` and maps each path to its respective handler method. Keeps routing declarative and separate from handler logic. |

### Models (Data Structures)

| File | Intent |
|------|--------|
| `models/profile.go` | Defines `CreatorProfile` — the GORM model for a creator's onboarding data (name, genre, language, region, tone, style, audience, content type, platform, experience, USP, etc.). This is the foundation for persona-based prompt generation. |
| `models/prompt.go` | Defines `Prompt` — stores a generated prompt linked to a creator and topic, with a `Variant` field (values: `"base"`, `"A"`, `"B"`, `"C"`) to support A/B testing of prompt styles. |
| `models/script.go` | Defines `Script` — stores the final AI-generated video script, linked to the source prompt and creator, with a `Source` field noting the model used (e.g. `"GPT-4"`). |
| `models/requests.go` | Request DTOs (`GeneratePromptRequest`, `GenerateScriptRequest`) used for JSON binding in handler endpoints. Separates wire format from persistence models. |

### Handlers (HTTP Layer)

| File | Intent |
|------|--------|
| `handlers/profile_handler.go` | Handles `POST /submit-profile` (validates JSON, calls `ProfileService.CreateProfile`, returns creator ID) and `POST /generate-prompt` (fetches profile, builds persona summary, generates a single prompt, persists it). |
| `handlers/prompt_ab_handler.go` | Handles `POST /generate-prompt-ab`. Fetches the creator's profile, generates three prompt variants (balanced, punchy, storytelling) via `PromptABService`, stores all variants, and returns them. |
| `handlers/prompt_handler.go` | Handles `GET /prompts/:creator_id`. Retrieves and returns all stored prompts for a given creator. |
| `handlers/script_handler.go` | Handles `POST /generate-script` (looks up the matching prompt, sends it to OpenAI via `ScriptService`, persists the result) and `GET /scripts/:creator_id` (lists all scripts with their source prompts). |
| `handlers/profile.go` | Empty stub — placeholder for future profile-related handlers. |
| `handlers/prompt.go` | Empty stub — placeholder for future prompt-related handlers. |

### Services (Business Logic)

| File | Intent |
|------|--------|
| `services/profile_service.go` | `ProfileService` interface + GORM implementation. Provides `CreateProfile` (persist new creator) and `GetProfileByID` (retrieve by primary key). |
| `services/prompt_service.go` | `PromptService` interface + implementation. Core logic: `GeneratePersonaSummary` (composes a natural-language persona from profile fields) and `GeneratePrompt` (wraps persona + topic into a structured prompt and saves it as variant `"base"`). |
| `services/prompt_ab_service.go` | `PromptABService` interface + implementation. Generates three prompt variants — **A** (balanced, informative), **B** (punchy, hook-driven), **C** (storytelling, emotional) — using the persona summary and topic, then stores them via the repository. |
| `services/prompt_variant_repository.go` | `PromptRepository` interface + GORM implementation. Provides `SavePrompt` (insert) and `GetPromptsByCreatorID` (list by creator). Decouples persistence from service logic. |
| `services/script_service.go` | `ScriptService` interface + `OpenAIScriptService` implementation. Makes HTTP POST calls to `api.openai.com/v1/chat/completions` with the prompt text, includes 3-retry logic with exponential wait, and parses the response into script text. |
| `services/script_repository.go` | `ScriptRepository` interface + GORM implementation. `SaveScript` (persist) and `GetScriptsByCreatorIDWithPrompt` (JOIN scripts + prompts to include prompt context in listing). |
| `services/persona.go` | Empty stub — placeholder for future standalone persona logic. |
| `services/prompt.go` | Empty stub — placeholder for future prompt utilities. |

### Utilities

| File | Intent |
|------|--------|
| `utils/secrets.go` | Loads sensitive configuration (`OPENAI_API_KEY`, `OPENAI_MODEL`) from environment variables into a typed `Secrets` struct. Single source of truth for external API credentials. |
| `utils/mapping.go` | Empty stub — placeholder for future data mapping/transformation helpers. |

### Configuration & Meta

| File | Intent |
|------|--------|
| `go.mod` | Go module definition (`github.com/ashisharyan/ghostwriter-prompt-engine`). Declares dependencies: Gin, GORM, SQLite driver, godotenv, Swagger. |
| `go.sum` | Dependency integrity checksums (auto-generated). |
| `.env.example` | Template for environment variables: `db_path`, `port`, `OPENAI_API_KEY`, `OPENAI_MODEL`. |
| `.gitignore` | Ignores build artifacts, `docs/` (Swagger), IDE files, OS files, logs. |
| `README.md` | User-facing documentation: setup instructions, API reference with request/response examples, project description. |
| `routes.http` | Empty HTTP client test file (for use with REST Client extensions). |

### Planning Documents (`planning/`)

| File | Intent |
|------|--------|
| `planning/central-planning-document.md` | Full product vision, MVP scope, target audience, tech stack decisions, revenue model. Describes the broader product (frontend, payments, delivery) beyond this backend. |
| `planning/gaps-and-questions.md` | Open design questions and unresolved decisions. |
| `planning/prompt.md` | Prompt engineering notes — how the system prompt for OpenAI should be structured. |
| `planning/user-stories-epic1-instagram-marketing.md` | User stories for Epic 1: Instagram marketing features. |
| `planning/user-stories-epic2-onboarding.md` | User stories for Epic 2: Creator onboarding flow. |
| `planning/user-stories-epic3-script-generation.md` | User stories for Epic 3: Script generation features. |
| `planning/user-stories-epic4-delivery.md` | User stories for Epic 4: Script delivery (WhatsApp, email). |
| `planning/user-stories-epic5-payments.md` | User stories for Epic 5: Payment integration (Razorpay). |
| `planning/user-stories-epic6-dashboard.md` | User stories for Epic 6: Creator dashboard. |

---

## Data Flow

### 1. Creator Onboarding
```
Client POST /submit-profile → ProfileHandler.SubmitProfile → ProfileService.CreateProfile → SQLite
```

### 2. Prompt Generation (Single)
```
Client POST /generate-prompt {creator_id, topic}
  → ProfileHandler.GeneratePrompt
    → ProfileService.GetProfileByID
    → PromptService.GeneratePersonaSummary (builds persona string from profile)
    → PromptService.GeneratePrompt (wraps persona+topic into prompt)
    → PromptRepository.SavePrompt → SQLite
```

### 3. A/B/C Prompt Generation
```
Client POST /generate-prompt-ab {creator_id, topic}
  → PromptABHandler.GeneratePromptAB
    → ProfileService.GetProfileByID
    → PromptABService.GeneratePromptVariants
      → Creates 3 variants (balanced / punchy / storytelling)
    → PromptRepository.SavePrompt (×3) → SQLite
```

### 4. Script Generation
```
Client POST /generate-script {creator_id, topic, variant}
  → ScriptHandler.GenerateScript
    → PromptRepository.GetPromptsByCreatorID (find matching prompt)
    → ScriptService.GenerateScriptFromPrompt
      → HTTP POST api.openai.com/v1/chat/completions (with retries)
    → ScriptRepository.SaveScript → SQLite
```

---

## Design Patterns

| Pattern | Where Used |
|---------|-----------|
| **Dependency Injection** | `cmd/main.go` wires all interfaces to concrete implementations |
| **Repository Pattern** | `PromptRepository`, `ScriptRepository` abstract persistence |
| **Service Layer** | Business logic isolated in `services/` behind interfaces |
| **Handler/Controller** | HTTP concerns isolated in `handlers/` |
| **Interface Segregation** | Small, focused interfaces (`ProfileService`, `PromptService`, `ScriptService`) |

---

## Known Issues & Technical Debt

1. **Type inconsistency** — `CreatorProfile.ID` is `uint`, but `Prompt.CreatorID` and `Script.CreatorID` are `string`. Conversion happens via `fmt.Sprintf`.
2. **Empty stubs** — 5 files contain only a package declaration (`handlers/profile.go`, `handlers/prompt.go`, `services/prompt.go`, `services/persona.go`, `utils/mapping.go`).
3. **Missing Swagger docs** — `main.go` imports a `docs` package that is gitignored and absent; requires `swag init` to generate.
4. **Env var casing mismatch** — `.env.example` uses lowercase (`db_path`, `port`) but code reads uppercase (`DB_PATH`, `PORT`).
5. **Committed binary artifacts** — `go1.21.6.linux-amd64.tar.gz` (~64MB) and `ghostwriter.db` are in the repo.
6. **No tests** — No test files exist in the project.
7. **No authentication** — All endpoints are publicly accessible with no auth middleware.
