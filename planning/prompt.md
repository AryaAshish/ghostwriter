🧠 VISION
We’re building an AI content generation tool for Instagram and YouTube creators in India that mimics their exact personal tone, identifies trending topics, and writes high-performing video scripts tailored to their niche and voice.

We want creators to feel:

“This sounds like I wrote it — but I didn’t have to.”

🎯 MVP GOAL (Phase 1)
Launch a lean product in 1–1.5 months that:

Acquires first 10–15 paying creators

Personalizes scripts using a Q&A-based onboarding form

Delivers 30 scripts/month in the user’s own voice

Includes basic payment integration

Builds awareness via Instagram-first content marketing

🧱 CORE COMPONENTS
Insta-first content page (Build trust & credibility)

Frontend onboarding flow (Collect niche + voice style)

Script generation engine (GPT-powered, prompt-tuned)

Trend integration (manual for now) (Daily topic injection)

Delivery system (Email/WhatsApp/Google Doc link)

Payment layer (Razorpay or Stripe)

Basic creator dashboard (optional in MVP)

📦 PRODUCT EPICS & USER STORIES
✳️ Epic 1: Instagram Page for Brand & Acquisition
Goal: Build early audience & warm up creator market.

User Stories:
 As a founder, I want to create an Instagram page targeting Indian creators so we can start building awareness.

 As a marketer, I want to post 1–2 daily carousels/reels showcasing trending script examples.

 As a team, I want to track engagement and DMs so we can identify potential beta users.

 As a growth hacker, I want to test creator meme-style posts and “If XYZ explained AI” content.

✳️ Epic 2: Script Personalization Onboarding
Goal: Collect data to personalize script outputs.

User Stories:
 As a user, I want to answer 5–6 style questions so the system understands my tone.

 As a user, I want to select my niche (tech, finance, lifestyle, etc.)

 As a user, I want to upload past scripts or link a Reel/Shorts video (optional)

 As a system, I want to store creator inputs to personalize future prompts.

✳️ Epic 3: Script Generation Engine (Prompt + AI)
Goal: Generate niche + tone-aligned scripts daily.

User Stories:
 As a user, I want to receive 30 scripts per month, aligned with my style and niche.

 As a backend dev, I want to create prompt templates per niche with tone variations (funny, sarcastic, educational).

 As a system, I want to inject daily trends into scripts (manual for now).

 As a user, I want to get 20 edit/rewrite prompts per script.

✳️ Epic 4: Delivery Mechanism
Goal: Seamlessly deliver personalized scripts to users.

User Stories:
 As a user, I want to receive my scripts via WhatsApp, email, or Google Docs link.

 As an admin, I want to queue & batch script delivery daily.

 As a user, I want a way to request script rewrite from the same place.

✳️ Epic 5: Payments & Trial Flow
Goal: Monetize through lean checkout.

User Stories:
 As a user, I want to pay ₹2,999/month for 30 scripts/month.

 As a system, I want to allow UPI + card via Razorpay/Stripe.

 As a user, I want to start with a ₹999 first-month trial.

 As admin, I want to track subscription users + failed payments.

✳️ Epic 6: Creator Experience Dashboard (Stretch Goal)
Goal: Let users view/edit scripts, update tone, and give feedback.

User Stories:
 As a user, I want to view all my scripts in one place.

 As a user, I want to thumbs up/down scripts to improve personalization.

 As a user, I want to edit or request rewrites within the dashboard.

 As a user, I want to update my tone preferences anytime.

⚙️ TECH STACK (SUGGESTED MVP)
Component	Stack/Tool
Frontend	Next.js / React / Vercel
Backend	Node.js / Supabase / Firebase
AI Layer	GPT-4 Turbo via OpenAI API
Data Storage	Supabase/Postgres DB
Delivery	WhatsApp (Twilio), Email, Docs
Payments	Razorpay / Stripe
Versioning	GitHub + Linear/Windsurf

🚀 PRIORITY ORDER FOR EXECUTION
Week	Focus
1–2	Insta launch + script prompt engine (manual trend injection)
2–3	Onboarding form + Q&A-based style capture
3–4	Script generation pipeline + delivery
4–5	Payments + trial offer
5–6	Onboard 10 paid creators

