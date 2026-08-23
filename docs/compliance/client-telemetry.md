# Client Telemetry Collection

## What We Collect
- Browser: user_agent, language, timezone, platform
- Screen: width, height, available dimensions
- Viewport: width, height
- Device: pixel ratio, color depth, touch points
- Client Hints: architecture, platform, mobile (where available)

## What We Do NOT Collect
- No canvas/WebGL/audio fingerprinting
- No font enumeration
- No keystroke/mouse tracking
- No clipboard/camera/microphone access
- No browser extension enumeration

## Privacy
- Telemetry is classified as optional analytics
- Integrated with existing cookie consent system
- Essential security logging continues regardless of consent
- Client never provides server-authoritative fields (IP, country, user_id)
