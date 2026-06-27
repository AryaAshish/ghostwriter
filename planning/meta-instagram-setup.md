# Meta / Instagram setup for Ghostwriter

Ghostwriter uses **one Meta Developer app** (yours) so creators can click **Connect Instagram**. Creators do **not** need a developer account — only you do.

## Prerequisites (creator side)

Each creator who connects must have:

1. **Instagram Business or Creator** account (not personal-only)
2. Instagram linked to a **Facebook Page** ([Meta instructions](https://www.facebook.com/business/help/898752960195806))

## 1. Create the Meta app (one-time, ~15 min)

1. Go to [developers.facebook.com](https://developers.facebook.com/) and log in.
2. **My Apps → Create App → Other → Business**.
3. Name it e.g. `Ghostwriter` and connect your Business portfolio (or create one).

## 2. Add Instagram product

1. In the app dashboard: **Add Product → Instagram → Set up**.
2. Use **Instagram API with Facebook Login** (Graph API).

## 3. OAuth settings

1. **App settings → Basic** — copy **App ID** and **App Secret**.
2. **Facebook Login → Settings** — add Valid OAuth Redirect URI:

   ```
   http://localhost:8080/api/v1/instagram/callback
   ```

   For production, add your real domain too:

   ```
   https://your-domain.com/api/v1/instagram/callback
   ```

## 4. Permissions

In **App Review → Permissions and Features**, request (or use in Development mode):

| Permission | Why |
|------------|-----|
| `instagram_basic` | Read profile + media |
| `pages_show_list` | Find Facebook Pages |
| `pages_read_engagement` | Access linked IG account |

In **Development mode**, only app admins/testers can connect — enough for Persona Lab testing.

## 5. Configure Ghostwriter

Copy `.env.example` to `.env` and set:

```env
META_APP_ID=your-app-id
META_APP_SECRET=your-app-secret
META_REDIRECT_URI=http://localhost:8080/api/v1/instagram/callback
APP_BASE_URL=http://localhost:8080

# Optional: transcribe reels when captions are short
OPENAI_API_KEY=your-openai-key
```

Restart the server:

```bash
go run ./cmd
```

Check status: `GET http://localhost:8080/api/v1/instagram/status` → `"configured": true`

## 6. Test in Persona Lab

1. Open http://localhost:8080/app/
2. Choose **Import from Instagram**
3. Click **Connect Instagram** → approve permissions
4. Select reels → **Prepare selected reels**
5. **Build persona**

## Transcription

- **Captions ≥ 25 words** → used directly (free)
- **Short/missing captions** → enable “Transcribe short captions” (uses OpenAI Whisper API; paid per minute)
- **No OpenAI key** → only captions are used

For fully free transcription, run [local Whisper](https://github.com/openai/whisper) offline and paste transcripts in the manual captions box.

## API endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/instagram/status` | Is OAuth configured? |
| GET | `/api/v1/instagram/auth-url` | Start connect flow |
| GET | `/api/v1/instagram/callback` | Meta redirect (do not call manually) |
| GET | `/api/v1/instagram/reels?session=…` | List reels after connect |
| POST | `/api/v1/instagram/prepare` | Resolve caption/transcript for selected reels |

## Troubleshooting

| Error | Fix |
|-------|-----|
| `instagram connect is not configured` | Set `META_APP_ID` and `META_APP_SECRET` |
| `no Instagram Business or Creator account` | Convert IG to Creator/Business + link Facebook Page |
| `instagram session expired` | Connect again (sessions last 1 hour) |
| Redirect URI mismatch | Redirect URI in Meta must match `META_REDIRECT_URI` exactly |

## Going to production

1. Switch Meta app to **Live** mode after App Review.
2. Set `APP_BASE_URL` and `META_REDIRECT_URI` to production URLs.
3. Add Privacy Policy URL in Meta app settings (required for review).
