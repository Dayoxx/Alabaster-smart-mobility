# Master Engineering Guide: Part 2

New Sections 21 to 29. These follow the upgrade file (`MASTER_ENGINEERING_GUIDE_UPGRADE.md`, which added 7A and 20). Add them to the Table of Contents of `MASTER_ENGINEERING_GUIDE.md`.

| # | Section |
|---|---|
| 21 | Bracelet build playbook (phases, gates, working without samples) |
| 22 | Bracelet threat model and security operations |
| 23 | Backend work list for bracelets |
| 24 | Mobile development (Flutter) |
| 25 | Product website |
| 26 | Knowledge library |
| 27 | Ownership, timeline and open items |
| 28 | Bracelet instalment plan (₦5,000 + ₦100, repaid ₦10 per commute) |
| 29 | Single-tier bracelet: Java Card in every bracelet (supersedes the Standard and Elite split) |

**How to read this guide.** Anything marked **Verify** is something I could not confirm from here (a library, a price, a regulation, a shipping route). Check it before you spend money or build on it. Brand and product names are placeholders where the public name is not decided.

---

## 21. Bracelet Build Playbook

> **Update:** there is now one bracelet quality for everyone, with a Java Card secure element in every unit. Where this section mentions Standard and Elite tiers, DESFire as the main chip, or Phase 6, Section 29 replaces it.

### 21.1 The principle

The bracelet changes the credential, not the money model. Fare engine, payment flow, idempotency, trips and settlement stay as Sections 5 and 8 describe. Build everything that does not need hardware first, then add hardware in small steps that each have a pass or fail test.

### 21.2 Phases and gates

Each phase has a gate. Do not start the next phase until the gate passes.

| Phase | Work | Gate |
|---|---|---|
| 0. No hardware (now) | Decisions D1, D2, D6. Migrations 0006 to 0008. `internal/credential` with a fake authenticator. `tap_service.go`. Mobile and website work (Sections 24 and 25). Cheap-tag shell experiments (21.4) | Fake bracelet taps produce correct trips, blocks and audit events in tests |
| 1. Samples and reader | Buy a reader and DESFire EV3 samples. Python spike. Settle the Random ID and key diversification question. Measure real tap time on a desk | You can authenticate to a spare chip from code, and you know which Go approach you will use for secure messaging |
| 2. Provisioning | `cmd/provision` on spare chips. Key derivation from a development master key. Register chips in the database as INACTIVE | A factory-fresh chip becomes a registered bracelet chip in one run, with an audit record |
| 3. Terminal integration | Android terminal app does start, step and finish against the real API. Measure end-to-end tap time | Tap under about one second on a good connection, and a clear message on a poor one |
| 4. Shell v1 | Print with a dummy inlay, tune and measure the coil, test on a table and a wrist, then reprint with the real chip | Reliable reads on a wrist at the angles drivers actually see |
| 5. Pilot | 10 to 20 bracelets, real drivers, small limits, close monitoring | A full operating day cash-free with no unexplained charge |
| 6. Elite | Java Card applet on development cards, same terminal, then Elite shell | Offline verification works, and an independent review is booked |

### 21.3 What you can do today with no chips

Backend and data:
1. Confirm which migrations are applied (Section 1 checklist).
2. Make decisions D1 and D2 in Section 20.
3. Write migrations 0006 (identity rename, audit log, UUID switch), 0007 (`credential`) and 0008 (`bracelet`).
4. Write `internal/credential/authenticator.go` and a fake authenticator that simulates: success, slow chip, malformed response, authentication failure, and a replayed tap.
5. Write `tap_service.go` and the three bracelet endpoints against the fake.
6. Write the tests in Section 15 and the bracelet additions to them.

Product:
7. Build the mobile screens against a fake backend (Section 24).
8. Build the website (Section 25).
9. Write the pilot plan: where, who, how many, what limits, who supports riders.

Hardware (cheap):
10. Buy inexpensive NFC tags (for example NTAG-type stickers or key-fob tags) and a few blank inlays. They do not have the security of DESFire, but they use the same 13.56 MHz coupling, so they let you test shell geometry, ferrite placement, wrist detuning and phone read range before the real chips arrive. Coil size and placement matter more here than the chip.
11. Print TPU test shells with a pocket for the tag. Measure and record what changes read range.

### 21.4 Getting samples and readers

**Verify** availability, minimum order and shipping to Nigeria before relying on any route.

- **DESFire EV3 chips:** ask NXP's authorized distributors (large electronics distributors list DESFire cards and inlays). Ask for sample cards in 4K or 8K. Distributor pages and NXP's own pages show current part numbers.
- **Readers:** start with a USB PC/SC reader that has documented EV2/EV3 support in the library you plan to use. A cheap reader may not support the commands you need. Check the library's documentation first, then buy the reader.
- **Marketplaces:** cheap chips from open marketplaces are sometimes counterfeit or cloned. DESFire EV3 supports an originality check that lets you verify a genuine NXP chip. Test every chip you buy this way before trusting it.
- **Import:** ask the seller about customs, delivery time and whether they ship NFC components to Nigeria. A freight forwarder is a common fallback.
- **Java Card (Elite):** ask distributors early. NDAs and minimum orders vary by vendor, and development cards often ship with documented default keys.

### 21.5 Hardware stack

Layers from the outside in:

1. Cover: 1 to 1.5 mm TPU or a non-metal plate
2. Coil and chip inlay, facing the terminal
3. Ferrite sheet behind the coil, to shield it from the wrist
4. Band material against the skin

Design rules:
- **No metal over the coil.** Your copper-plate render needs a metallic-finish plastic, a coating, or a metal ring that surrounds the coil rather than covering it. Test any real metal with and without ferrite and record the read range.
- **No glowing edge without power.** A passive bracelet has no battery. Options are a printed or photoluminescent edge that only looks lit, or a small LED powered by the reader's field that flashes on tap (it costs range, so test it). Do not add a battery for the glow.
- **Seal the pocket.** Sweat and rain are the main failure modes. Test with a sweat and rain exposure routine, and check that the chip still reads afterwards.
- **Size range.** Print S, M and L, or use an adjustable strap.

Tools: a Bambu Lab A1 mini (or any direct-drive printer good with TPU), 95A TPU filament, calipers, a NanoVNA for coil resonance at 13.56 MHz, ferrite sheet, and a phone with NFC for quick checks.

Print starting points (tune from here): direct drive, dry filament, slow speeds (about 20 to 30 mm/s), 0.4 mm nozzle, 0.2 mm layers, low retraction, a pause in the print at the layer where the pocket is open so you can insert the inlay.

### 21.6 Provisioning (Standard tier)

Provisioning prepares a blank chip. Order of steps:

1. Authenticate to the factory-fresh chip with its default key.
2. Change the chip master key to the bracelet's diversified key.
3. Create your application and set its keys.
4. Create the files: a small data file holding the bracelet ID, and a transaction MAC file where your reader library supports it.
5. Make the bracelet ID readable without authentication if your design needs it (see the Random ID question below).
6. Enable Random ID last. It cannot be undone, so practise on spare chips.
7. Register the chip in the database as INACTIVE and ship it INACTIVE.

**The Random ID question.** Key diversification uses chip-specific data, but Random ID hides the UID until after authentication. The server needs to know which key to try before authenticating. Common solutions are a freely readable file holding a non-secret bracelet ID, or deriving the key from that ID instead of the UID. Settle this on spare chips and write the answer in this guide.

**Verify** which Go approach works for EV2/EV3 secure messaging: an existing library, wrapping NXP's reader library, or implementing the AES-CMAC secure messaging layer yourself from NXP's documents. Decide after Phase 1.

### 21.7 Test plan (from the bracelet guide, with pass criteria)

| Test | Pass looks like |
|---|---|
| Read range on a table | Reliable at the distance a driver's hand will be |
| Read range on a wrist | Still reliable, with the loss versus the table written down |
| Awkward angles (window, door) | Taps work at angles drivers actually see |
| End-to-end tap time | Under about one second |
| Bend and flex | Still reads after repeated flexing |
| Sweat and rain | No failure after exposure |
| Rogue reader | A generic reader gets nothing useful |
| Replay | A captured tap is rejected |
| Freeze | A block or off request declines within a couple of seconds |
| Offline terminal | A cached blocklist works, and a stale cache forces a sync |

### 21.8 Pilot plan outline

- Scope: 10 to 20 bracelets, one corridor, drivers who volunteer, a named support person.
- Limits: low daily and unsettled caps, raised only after a clean week.
- Monitoring: daily review of declined taps, duplicate-tap events, latency, and any dispute.
- Rider support: a way to block a bracelet within minutes, including from a call or message to support staff.
- Exit criteria: no double charge, no unexplained charge, tap time acceptable, drivers willing to continue.
- Rollback: cash or the existing card path stays available during the pilot.

---

## 22. Bracelet Threat Model and Security Operations

### 22.1 Threats and defenses

| Threat | What it looks like | Defense |
|---|---|---|
| Clone a bracelet | Copy identifying data to another chip | Per-bracelet keys and authenticated reads. Never accept the UID alone as proof |
| Replay | Record a valid tap and send it again | Server-issued challenge, nonce replay check, duplicate-tap check |
| Relay | Forward a tap from a victim's wrist to a distant terminal in real time | Proximity Check where supported, short timeouts, server-side velocity checks (the same bracelet cannot tap in two places) |
| Skimming | Read a bracelet from a distance | Keep the design short range, authenticate before releasing anything useful, no sensitive data readable without authentication |
| Rogue terminal | A fake or stolen terminal produces charges | Terminal binding, signed requests, rate limits, kill switch per terminal |
| Insider at provisioning | A staff member copies key material | Master key in KMS or HSM, provisioning on a locked machine, audit of every run, two-person control for key operations |
| Master key leak | The key from which all bracelet keys derive is exposed | Key versioning (`key_version`), a documented rotation plan, and detection of unusual tap patterns |
| Lost or stolen bracelet | Someone else uses it | Instant block from the app and from support, limits on every bracelet, tap notifications |
| Social engineering at an agent | An attacker talks an agent into issuing or reissuing | Identity checks, agent actions logged, limits on how many an agent can issue |
| Platform bug | A logic error allows a double charge | Idempotency, concurrency tests, reconciliation |

### 22.2 Operations

- **Rotation plan.** Write down, before the pilot, how to move bracelets to a new master key. One option is to rotate lazily: when a bracelet next taps and authenticates, the server changes the chip's key to one derived from the new master key and updates `key_version`. This depends on being able to authenticate with the old key, so it does not help if the old key is already in an attacker's hands. In that case the response is to block and replace.
- **Incident response.** For a suspected key leak: suspend affected credentials, stop provisioning, review taps since the suspected time, notify affected riders, rotate, and write a review.
- **Two-person rule** for creating a master key, rotating it, or changing KMS permissions.
- **Backups.** Back up key material in a way that preserves the same access controls as the original. A backup with weaker controls defeats the purpose.
- **Outside review.** Book an independent security review before the pilot goes beyond a handful of friendly users, and before Elite launches. Chip certification does not cover your own applet or your own server code.

---

## 23. Backend Work List for Bracelets

| # | Task | Done when |
|---|---|---|
| 1 | Migration 0006: identity rename, polymorphic audit log, UUID key switch | `\dt` shows the new tables, old code compiles |
| 2 | Migration 0007: `credential` | Types, states, `chip_uid` unique, version column |
| 3 | Migration 0008: `bracelet` | Tier, edition, key_version, provisioned_at |
| 4 | `trips.credential_id` nullable column | Dispute lookups work |
| 5 | `internal/models/credential.go`, `bracelet.go` | Mirrors the tables |
| 6 | `credential_repository.go` | State changes use version checks and write an audit row in one transaction |
| 7 | `internal/credential` interface and fake | Fake simulates the failure cases in 21.3 |
| 8 | `tap_service.go` | Authenticates, then calls `payment_service` |
| 9 | Bracelet endpoints (start, step, finish) | Short-lived tap sessions, timeouts, rate limits |
| 10 | Credential lifecycle endpoints | Activate, deactivate, block, suspend (staff), reissue, upgrade |
| 11 | Notification hook | A tap or state change creates a message for the rider |
| 12 | Terminal blocklist endpoint | Sequence numbers so terminals can tell whether they are up to date |
| 13 | `cmd/provision` | After Phase 1 |
| 14 | Admin views | Search a bracelet, see state history, suspend, block, replace |
| 15 | Tests | Everything in Section 15 plus the bracelet additions |

---

## 24. Mobile Development (Flutter)

### 24.1 What exists in the plan

From the master guide: one Flutter codebase, feature-first folders, Riverpod for state, a signing network interceptor, `flutter_secure_storage` for sensitive local data, four logins (Driver, Passenger, Student, and a monitoring-only Admin). Privileged admin actions stay on the React web dashboard. Android first. iOS later.

For the bracelet, two apps matter most: the **driver terminal app** (reads the bracelet) and the **passenger app** (manages the bracelet).

### 24.2 Driver terminal app

**Purpose.** Start a shift, show the fare, read a bracelet, relay the chip to the server, show a clear result.

**Screens.**
1. Login and shift start (choose route, confirm terminal binding)
2. Ready screen: fare, route, network status, last result
3. Tap in progress: a simple animation and a timer
4. Result: APPROVED, DECLINED (with a short reason the driver can say aloud), or TRY AGAIN
5. Trip history and daily ticket status
6. Terminal status and support contact

**NFC approach.**
- Use Android's reader mode so the app reads a bracelet while it is in the foreground, without opening other apps. Use IsoDep to send raw commands to the chip.
- A Flutter NFC plugin that supports raw command exchange may work (for example `flutter_nfc_kit`). **Verify** it supports what you need (reader mode flags, timeouts, extended-length commands). If it does not, write a small Kotlin platform channel around `NfcAdapter.enableReaderMode` and `IsoDep`. A platform channel gives the most control over timing, which matters for a tap that must finish in about a second.
- Keep the NFC code in one class behind an interface (`ChipTransport`), with a fake implementation that replays recorded exchanges. This lets you test the whole app without a real bracelet.

**Tap flow in the app.**
1. Detect a chip
2. Call `POST /tap/bracelet/start`
3. Send the returned command to the chip
4. Send the chip's reply to `POST /tap/bracelet/step`
5. Repeat until the server says done
6. Call `POST /tap/bracelet/finish`
7. Show the result

**Performance rules.**
- Reuse one HTTP connection, keep request bodies small, and set short timeouts.
- Measure each step: chip time, network time, server time. Log them (without chip data) so you can see where time goes.
- Show a clear message when the network is poor. Do not let a driver wait without feedback.
- Offline is not available for the Standard tier because keys live on the server. The app must say so plainly.

**Budget phone considerations.**
- Test on the cheapest Android devices you plan to deploy. NFC antenna position and strength differ a lot between models, and some low-cost phones have weak NFC.
- Test with the screen dim, battery saver on, and mobile data only.
- Keep the app light: no large images on the tap screen, no heavy animation.

### 24.3 Passenger app

**Screens.**
1. Sign up and identity step (as your enrollment flow requires)
2. Link bank accounts (up to three, with priority order)
3. **Claim a bracelet:** scan a QR code or enter a code printed in the packaging. Using a code is simpler than reading the bracelet with the phone, and it also works on phones without NFC. The bracelet remains INACTIVE until claimed and activated.
4. Bracelet screen: state, ON and OFF switch, Block, last taps, replacement status
5. Trip and payment history
6. Disputes: "I did not make this tap"
7. Help and support contact

**Notifications.**
- Push a message for each tap (amount, route, time), each state change, and each failed or declined payment. Use a standard push service for Android. Tap alerts are a basic safety feature and belong to every tier.
- Provide an SMS fallback for riders without data.

**Security on the phone.**
- Store tokens in `flutter_secure_storage`. Never use plain preferences.
- Require re-authentication for Block-reversal-type actions and for linking accounts.
- Do not show full account numbers. Show the last four digits.
- Screens that show sensitive data should use the platform's screen-capture protection where it makes sense.

### 24.4 Student app and admin monitoring

Follow the existing plan: Student is a login mode with its own fare rules, and the mobile Admin is monitoring-only. Nothing privileged goes in the mobile app.

### 24.5 Project setup and quality

- **Structure:** feature-first folders as in Section 12. Define each repository interface before the screen, build the screen last.
- **State:** Riverpod providers per feature. Widgets never call the network directly.
- **Networking:** one client with signing, retry rules and timeouts. Idempotency keys on any request that can cause a charge.
- **Environments:** dev, staging and production flavors with different base URLs and different keys. Staging uses the Moniepoint sandbox.
- **Testing:** unit tests for providers and repositories, widget tests for each screen state (loading, success, error, offline), and integration tests that run the whole tap flow against the fake chip transport.
- **Accessibility and language:** large tap targets, high contrast, plain language. Consider more than one language for rider-facing screens, chosen by what your riders actually use. This is a suggestion to validate with the pilot riders.
- **Release:** a Google Play developer account, app signing, internal testing track first, then a closed pilot track. Keep an in-app minimum-version check so you can retire an old app that has a bug.
- **Crash and error reporting:** add crash reporting from the first pilot build, with the same never-log rules as the backend.

### 24.6 Mobile build order

1. Project skeleton, flavors, network client, secure storage
2. Auth screens against a fake backend
3. Passenger: bracelet screen and history against the fake
4. Driver: shift start and ready screen
5. Driver: tap flow with the fake `ChipTransport`
6. Connect both to the real staging API
7. Real NFC with a sample chip (after Phase 1)
8. Measure on budget phones, then tune
9. Pilot build

---

## 25. Product Website

### 25.1 Purpose and audiences

One site, several audiences:
- **Riders:** what the bracelet is, how to get one, how it stays safe
- **Drivers and unions:** how fares and payouts work, daily ticket, support
- **Partners and investors:** the problem, the pilot, the team
- **Press and regulators:** facts, contact and policies

### 25.2 Pages

| Page | Content |
|---|---|
| Home | One clear sentence on what it does, the bracelet image, a short "how it works" in three steps, a call to join the waitlist |
| How it works | Tap, fare, payment, receipt. A short diagram. Explain that the bracelet holds no money |
| For riders | Getting a bracelet, linking an account, turning it off, blocking it if lost |
| For drivers | Starting a shift, fares, payouts, the daily ticket, support |
| The bracelet | Standard and Elite tiers, what is the same for everyone (safety features), colorways |
| Safety and privacy | What data you hold, what you do not hold, how a lost bracelet is blocked, how disputes work |
| Pilot | Where it is running, who can join, what it covers |
| About | Team and mission |
| FAQ | Short honest answers, including what happens if the network is down |
| Contact and waitlist | Form, plus a messaging app link if your riders prefer it |
| Privacy policy and terms | Required before collecting any personal data |

### 25.3 Content rules

- **Be accurate, not absolute.** Do not claim "unhackable", "100% secure" or "works offline" for the Standard tier. Say what the bracelet does and what protections exist.
- **Label concept images as concepts** until real prototypes exist. Your renders are good for the home page, but they should say "concept" until a printed unit is photographed.
- **Do not publish pilot details that could embarrass a partner** before you have their agreement.
- **Payment and licensing claims.** Do not describe your regulatory status until you have confirmed it with a licensed provider or adviser (see D2 and the regulation risk in 7A.5).
- **Elite tier.** Do not advertise it as available until it is.
- **Brand name.** The public product name is separate from the internal project name. Decide it before the site goes live.

### 25.4 Privacy and data

- Publish a privacy policy that matches what you collect. Nigeria has a data protection law (the Nigeria Data Protection Act, 2023) and a regulator. **Verify** your obligations with a Nigerian data protection adviser before launch, including whether you must register as a data controller.
- Collect only what the waitlist needs: name, phone or email, area, role (rider, driver, partner).
- Store waitlist data in your own database, not in an untracked spreadsheet.
- Give a way to ask for deletion.
- If you use analytics, choose a privacy-respecting one and say so.

### 25.5 Technical approach

You already work in React and TypeScript, so a static-first React framework is a sensible fit. Options to compare: Next.js or Astro (static output with a little interactivity). Keep it simple.

- **Performance first.** Many visitors will be on mobile data. Compress images, use modern formats, lazy-load below the fold, and keep total page size small. Aim for fast loading on a mid-range phone on a slow connection.
- **Hosting.** A static host with a CDN (several free or low-cost options exist). Put the site on its own domain. Keep the API on a separate subdomain.
- **Waitlist form.** Post to a Go endpoint on your API with validation, rate limiting and a spam check. Do not trust the browser. Never put API secrets in the site's code.
- **Security.** HTTPS everywhere, a Content Security Policy, and no third-party scripts you do not need.
- **SEO and sharing.** Page titles, descriptions, a sitemap, and share images so links look right when sent on messaging apps.
- **Accessibility.** Proper headings, alt text, readable contrast, keyboard navigation.
- **Contact options.** Many riders prefer messaging apps over email, so offer a messaging link where it suits your audience.

### 25.6 Website build order

1. Decide the public name and tone
2. Write the copy for Home, How it works and Safety
3. Build the layout and the three core pages
4. Add the waitlist form and backend endpoint
5. Add the remaining pages
6. Privacy policy and terms (after advice)
7. Performance, accessibility and security pass
8. Launch with a waitlist, then update as the pilot proves things

---

## 26. Knowledge Library

Study by track. Items marked **(core)** are worth reading in full. Anything online may have moved, so search the title.

### 26.1 NFC, smart cards and DESFire

- **ISO/IEC 14443** (parts 1 to 4): contactless cards. Read the overview of parts 3 and 4 first, then the APDU transport.
- **ISO/IEC 7816-4**: command and response format (APDUs). **(core)**
- **NXP's MIFARE DESFire EV3 data sheet and application notes** (search the NXP website): the key features, secure messaging, transaction MAC, Proximity Check. **(core)**
- **NXP AN10922**: symmetric key diversification, which the bracelet guide uses.
- **Smart Card Handbook** (Rankl and Effing): broad and practical.
- **RFID Handbook** (Finkenzeller): antenna and coupling theory, useful for coil design.
- **NFC Forum** public specifications: background on how readers and tags work.
- **Tools for inspection:** pyscard (PC/SC from Python), and a Proxmark3 for analyzing your own chips and reader behavior. Use these only on chips and systems you own.

### 26.2 Java Card (Elite)

- Oracle's Java Card documentation and development kit.
- GlobalPlatformPro for loading applets.
- GlobalPlatform card specification (overview): how applets are installed and secured.
- Elliptic curve signatures (ECDSA) and certificate chains, so you can design your CA.
- An independent reviewer's checklist for applets (ask the reviewer before you write the applet).

### 26.3 Cryptography and security

- **Serious Cryptography** (Aumasson): readable, practical. **(core)**
- AES, CMAC, challenge-response, and why you do not invent your own scheme.
- Key management: KMS and HSM concepts, key rotation, separation of duties.
- **OWASP API Security Top 10.** **(core)**
- OWASP Mobile Application Security guidance, for the Flutter apps.
- Threat modeling basics (use the table in 22.1 as a start and extend it).

### 26.4 Go backend and databases

- **Learning Go** (Bodner), plus the official Go documentation, tour and effective Go.
- Go `context`, graceful shutdown, `log/slog`, and testing with fakes.
- **pgx/v5** documentation: pooling, transactions, error handling.
- PostgreSQL documentation: transactions and isolation levels, locking (`SELECT ... FOR UPDATE`), indexes, constraints, `EXPLAIN ANALYZE`.
- **Designing Data-Intensive Applications** (Kleppmann): the chapters on transactions, consistency and failure. **(core)**
- Idempotency keys and retry design (search for provider documentation and engineering write-ups).
- Migration tools: goose or golang-migrate documentation.
- PostGIS documentation for the geospatial layer.

### 26.5 Payments and regulation (Nigeria)

- Moniepoint's developer documentation: authentication, transfers, webhooks, status lookup, idempotency, sandbox.
- Central Bank of Nigeria (CBN) guidance on payments and the rules for payment service providers. **Verify** with a licensed provider or a Nigerian fintech lawyer which licenses your model needs.
- NIBSS and NIP rails documentation for how transfers settle.
- The Nigeria Data Protection Act, 2023 and guidance from the regulator.
- Reconciliation practice: how to match your ledger to provider statements.
- **Verify** whether radio equipment approvals apply to your terminals or bracelets with a regulatory adviser.

### 26.6 Mobile (Flutter and Android)

- Flutter documentation: layout, navigation, platform channels, testing, flavors.
- Dart language tour.
- Riverpod documentation.
- Android developer documentation: NFC basics, `NfcAdapter` reader mode, `IsoDep`, and host card emulation for the later phone credential.
- `flutter_secure_storage` and Android Keystore concepts.
- Push notification service documentation for Android.
- Play Console documentation: signing, tracks, release management.
- Performance on low-end devices: profiling and startup time.

### 26.7 Hardware and manufacturing

- Antenna basics at 13.56 MHz: resonance, quality factor, mutual coupling, detuning.
- NanoVNA user guide and tutorials.
- Printing TPU: speeds, retraction, drying, bridging.
- Design for manufacture: when you leave printing for injection molding or overmolding (silicone or TPU), minimum order quantities, tooling cost and lead times. Talk to two or three contract manufacturers before choosing a shell design you cannot make at scale.
- Skin-contact materials: ask suppliers for material data on skin safety and durability.
- Incoming quality control: an originality check and a read test on every chip batch.

### 26.8 Web and product

- Your chosen framework's documentation (Next.js or Astro).
- Web performance basics (image formats, lazy loading, caching).
- Web accessibility guidelines (WCAG).
- Content Security Policy.
- Writing plain-language product copy for a mixed audience.

### 26.9 Suggested study order

1. **Now:** Go and PostgreSQL transactions, idempotency, the Section 15 tests, Flutter basics.
2. **Before the reader arrives:** ISO 7816-4 APDUs, DESFire EV3 data sheet, AN10922, pyscard.
3. **While building provisioning:** key management and the AES and CMAC material.
4. **While building the terminal app:** Android NFC reader mode and IsoDep.
5. **Before the pilot:** OWASP API Top 10, reconciliation, threat model review, Nigerian regulation.
6. **Later:** Java Card, ECDSA and certificates, design for manufacture.

---

## 27. Ownership, Timeline and Open Items

### 27.1 Suggested tracks

You have co-founders, so assign an owner to each track. Names are placeholders.

| Track | Owner | First deliverable |
|---|---|---|
| Backend and payments | | Idempotency (#5), migrations 0006 to 0008 |
| Bracelet hardware | | Cheap-tag shell experiments, then samples and reader |
| Mobile apps | | Passenger bracelet screen and driver tap screen against the fake |
| Website and brand | | Public name, copy, and a waitlist page |
| Regulation and company setup | | Written advice on the money model (D2) and data protection |
| Pilot operations | | Corridor, driver recruitment, support plan |

### 27.2 Rough order, not dates

I have not put dates here because your timeline depends on hardware lead times and regulatory advice that you do not control. Order the work as in 21.2 and set dates after Phase 0 is done and you have a quote and delivery time for the first chips.

### 27.3 Open items

| Item | Needed before |
|---|---|
| D1 grouped payments scope | Phase 3 |
| D2 money model and licensing | Any settlement code, and before the website describes payments |
| D4 phone credential alongside physical | Phone credential work |
| D5 database driver check | Next repository change |
| D6 identity rename and UUID switch | Migration 0007 |
| Public product name | Website launch |
| Random ID and key derivation approach | Provisioning |
| Go approach for DESFire secure messaging | End of Phase 1 |
| Source and shipping route for chips and readers | Phase 1 |
| Data protection registration and policy | Website waitlist launch |
| Equipment approval question | Pilot |
| Security review booking | Before the pilot widens |

### 27.4 Things I could not verify from here

- Which Go library, if any, supports DESFire EV2/EV3 secure messaging
- Whether `flutter_nfc_kit` fits your needs on your target phones
- Current distributor stock, minimum orders and delivery to Nigeria
- Which licenses your payment model requires
- Whether radio equipment approval applies
- Your obligations under the data protection law

Treat each as a research task with a named owner, not as an assumption.

---

## 28. Bracelet Instalment Plan

### 28.1 The model

The bracelet is sold on instalments. The rider takes the bracelet without paying the full price up front, and a small amount is added to each fare until the bracelet is paid off.

| Item | Naira | Kobo (what the database stores) |
|---|---|---|
| Bracelet price | ₦5,000 | 500,000 |
| Instalment fee | ₦100 | 10,000 |
| Total to repay | ₦5,100 | 510,000 |
| Deducted per commute | ₦10 | 1,000 |
| Commutes needed | 510 | |

**How long repayment takes** depends on how often the rider travels:

| Rides per day | Days to repay | About |
|---|---|---|
| 1 | 510 | 17 months |
| 2 | 255 | 8.5 months |
| 4 | 128 | 4 months |
| 6 | 85 | 3 months |

A rider who travels rarely may take well over a year, and a rider who stops travelling never finishes. That is the central risk of this model.

**Assumption to confirm (D7):** the ₦10 is charged on top of the fare and on top of the existing passenger platform fee, so the rider's total for one ride is fare + platform fee + ₦10. If it is meant to come out of the platform fee instead, the revenue numbers change and the rider's total does not.

### 28.2 How this differs from decision D2

D2 asked whether the platform approves a tap before it has collected the fare. The instalment plan is a different exposure. The platform gives the hardware first and collects over time. The fare itself is still collected at the tap through Moniepoint, so the Section 5.8 invariant (only `payment = SUCCEEDED` makes a fare successful) is unchanged. But the platform now holds a receivable from each rider, and it carries the risk that some riders never repay.

### 28.3 Numbers to check before launch

This is a business question, and it needs your own figures:

- **Unit cost.** What does one bracelet cost you in chip, shell, printing or molding, packaging, provisioning and delivery? If that is close to ₦5,000, the plan has no room for losses.
- **The ₦100 fee.** It is a flat 2% of the price, whether the rider repays in three months or seventeen. Over a long repayment period, inflation and your own cost of money may be larger than that. Model it at several repayment speeds.
- **Default rate.** Estimate how many riders will stop using the bracelet before they finish. Run the numbers at 5%, 15% and 30%.
- **Pilot exposure.** For 20 bracelets, the most the platform can be owed is 20 × ₦5,100 = ₦102,000, and the hardware is already spent. That is small enough to learn from. Set a ceiling on total outstanding balance so the exposure does not grow faster than the evidence.

### 28.4 Decisions to make

| ID | Decision | Options |
|---|---|---|
| D7 | Is the ₦10 on top of fare and platform fee? | On top (assumed), or taken out of the platform fee |
| D8 | Lost or stolen bracelet | The balance stays with the rider's account and does not disappear. Decide whether a replacement costs extra and how many replacements one rider may have |
| D9 | Rider stops travelling | Options include a small refundable deposit or activation fee, a deadline after which the rest falls due, or a limited number of bracelets per rider until the first is paid off. Choose before launch |
| D10 | Early payoff and returns | Allow paying the remaining balance early, and decide whether a returned bracelet gets a refund of what was paid |
| D11 | Regulation and tax | Selling goods on instalments can count as consumer credit. Check consumer protection and digital lending rules (for example the FCCPC), and how VAT and revenue are treated. Get written advice |
| D12 | Ownership until paid | State in the terms who owns the bracelet until the balance is cleared, and what the platform can and cannot do about an unpaid balance |
| D13 | Limits | Maximum unpaid bracelets per identity, and per agent or location |

**Verify** D11 with a Nigerian fintech lawyer or adviser before you sell a single bracelet this way. Do not assume the rules that apply to your fare payments also cover a payment plan on hardware.

### 28.5 Data model

One plan per bracelet purchase, and a ledger of everything that changes the balance. Store the terms on the plan itself, so a later change to the price or fee never alters an existing plan.

```
bracelet_plan
  id                 uuid PK
  identity_id        uuid  -- the account that owes the balance (not the bracelet)
  credential_id      uuid  -- the bracelet that was sold; may change on replacement
  price_amount       bigint   -- 500000 at the time of sale
  fee_amount         bigint   -- 10000
  total_amount       bigint   -- price + fee
  per_tap_amount     bigint   -- 1000
  paid_amount        bigint   -- running total, never above total_amount
  status             text     -- ACTIVE | PAID | CANCELLED | WRITTEN_OFF
  terms_version      integer  -- links to the versioned configuration
  version            integer  -- optimistic lock
  created_at, updated_at  timestamptz

bracelet_plan_entry
  id                 uuid PK
  plan_id            uuid FK
  kind               text     -- DEDUCTION | REVERSAL | MANUAL_PAYMENT | ADJUSTMENT
  amount             bigint
  payment_id         uuid NULL -- the payment attempt that carried it
  trip_id            uuid NULL
  created_at         timestamptz
  UNIQUE (plan_id, payment_id, kind)   -- one deduction per payment
```

Constraints:
- `CHECK (paid_amount >= 0 AND paid_amount <= total_amount)`
- Amounts are integer kobo, never floating point (Section 5.1)
- The balance is the sum of the entries. The `paid_amount` column is a cache that is updated in the same transaction as the entry, guarded by the `version` column
- Plan status changes write an audit event (privileged-mutation pattern, Section 10)

In `payment_allocation`, add a purpose so each part of a payment is labeled: `FARE`, `PLATFORM_FEE`, `INSTALMENT`.

### 28.6 How a tap changes

1. **The fare quote includes the instalment.** The server computes `instalment_amount = MIN(per_tap_amount, total_amount - paid_amount)` for the rider's active plan. It is never accepted from the terminal or the app (Golden Rule on server-controlled fare).
2. **Passenger total** = fare + passenger platform fee + `instalment_amount`.
3. **The instalment counts only when the payment succeeds.** Create the plan entry in the same database transaction that marks the payment `SUCCEEDED` and the trip `COMPLETED`.
   - Payment `FAILED`: no entry.
   - Payment `UNKNOWN`: no entry yet. If reconciliation later confirms success, create the entry then.
   - Refund or reversal of the payment: create a `REVERSAL` entry for the instalment part.
4. **The last payment is capped** so the rider never pays more than the total. When the balance reaches zero, set the plan `PAID` and stop adding the line.
5. **Concurrency.** Two taps at once near the end of a balance must not overpay. Lock the plan row or use the version check, and test it (28.9).
6. **Receipt and history** show the instalment as its own line, with the remaining balance.
7. **Disputes.** If a rider successfully disputes a tap, the instalment part is reversed with the rest.

The driver's payout, the daily ticket and the fare engine are not affected. The instalment is a passenger-side line that goes to the platform.

### 28.7 Upgrade, replacement and tiers

- The balance belongs to the account, not the chip. If the bracelet is replaced or upgraded, point the plan at the new credential and keep the balance.
- A block does not erase the balance. A blocked bracelet cannot be used for taps, so no instalments are collected until a replacement is active. Decide in D8 how that interacts with any fee for replacements.
- If the rider upgrades to Elite, decide whether the rest of the Standard balance is paid off at that point or carries into a new plan.
- Do not use the `SUSPENDED` state as a collection tool without a written policy. It is for security. Using it to pressure a rider over a balance is a regulatory and reputation risk, and D11 advice should cover it.

### 28.8 App, website and operations

**Passenger app:**
- A bracelet balance card: paid so far, remaining, a progress bar, and roughly how many rides are left at their usual pace
- A line on every receipt
- A "pay off early" action
- A clear statement of the terms they agreed to, available at any time

**Website and packaging:** state the terms in plain words, for example "The bracelet costs ₦5,000. You pay ₦10 extra on each ride until you have paid ₦5,100 in total. You can pay the rest early at any time." Do not use the word "free" for a bracelet that is being repaid.

**Consent:** record that the rider accepted the terms (version, time, how they accepted) before activation.

**Admin and reports:**
- Total outstanding balance, and balance by age
- Number of bracelets with no taps in 30 days, 60 days and 90 days
- Repayment rate against what you modeled
- A daily check that instalment entries equal the instalment allocations of succeeded payments
- Alerts when outstanding balance nears the ceiling you set

### 28.9 Tests

- A successful tap adds exactly one entry of ₦10 and the receipt shows it
- A failed payment adds no entry
- An unknown payment adds no entry until reconciliation confirms it
- Retrying the same payment never adds a second entry
- The 510th deduction brings the balance to exactly zero, and the 511th tap has no instalment line
- A tap with ₦4 remaining deducts only ₦4
- Two taps at once with ₦10 remaining produce exactly one deduction
- A reversal restores the balance
- Replacing the bracelet keeps the balance and moves the plan to the new credential
- Changing the configured price or fee does not change an existing plan
- A rider cannot exceed the limit of unpaid bracelets
- Early payoff closes the plan and stops the instalment line

### 28.10 Changes to earlier sections

| Section | Change |
|---|---|
| 4, #20 | Bracelet price, instalment fee and per-tap amount are versioned configuration. Each plan keeps the terms it was sold under |
| 5.1 | Add `instalment_amount` to the trip financial fields, and include it in `passenger_total` and `platform_total` |
| 6 | Add a purpose to `payment_allocations` (FARE, PLATFORM_FEE, INSTALMENT) |
| 7.6 | Replacement and upgrade keep the instalment balance on the account |
| 7A.2 | Add the plan tables to the bracelet migrations |
| 14 | Add instalment reports and the daily consistency check |
| 15 | Add the tests in 28.9 |
| 20 | Add decisions D7 to D13 |
| 21.8 | The pilot runs with a ceiling on total outstanding balance |
| 25 | Website states the terms in plain words |

### 28.11 Build order

1. Make D7, then D8 to D13 with your co-founders and an adviser
2. Model the numbers in 28.3
3. Add the versioned configuration for price, fee and per-tap amount
4. Migrations for `bracelet_plan` and `bracelet_plan_entry`, and the allocation purpose column
5. Fare quote with the instalment line, then the entry written inside the payment-success transaction
6. Tests in 28.9, including concurrency
7. Passenger app balance card and receipts
8. Admin reports and the daily check
9. Terms text, consent record and website wording
10. Pilot with a ceiling on exposure

---

## 29. Single-Tier Bracelet: Java Card in Every Bracelet

### 29.1 The decision

There is no premium tier. Every bracelet gets the same chip, the same applet, the same security and the same service. Only cosmetic editions (colorway, numbered runs) may differ, and they do not change price or protection.

This replaces the Standard (DESFire) and Elite (Java Card) split in Section 2 of the original bracelet build guide and in Sections 7A and 21 of this guide. DESFire stays on the list only as a fallback and for a possible low-cost or phone-based credential later.

Limits (daily and unsettled caps) are set by rider risk and history, not by product tier. A new rider starts low and the limit rises with a clean record (D18).

### 29.2 What changes

| Topic | Before (two tiers) | Now (Java Card for all) |
|---|---|---|
| Chip | DESFire EV3 for most, Java Card for Elite | Dual-interface Java Card secure element in every bracelet |
| Keys | AES keys derived on your server | Each bracelet generates its own private key on the chip. It never leaves the chip |
| What the server stores per bracelet | A way to derive its key | The bracelet's public key and certificate serial |
| Server breach | Could expose how card keys are derived | Cannot reveal any bracelet's private key |
| Tap protocol | Several network round trips relaying chip messages | A few messages between terminal and chip locally, then one network request |
| Latency risk | High, because every chip message crossed the network | Much lower |
| Random ID question | Open | Gone, because the card is identified by its certificate |
| Offline | Not possible | Possible for identity checks, with limits (29.3) |
| Applet | None | You need one, and it carries security risk |
| Sourcing | Open distributor samples | Vendor NDAs, minimum orders, development cards |
| Unit cost | Lower | Higher. Get quotes |
| Review | Server code and provisioning | Also the applet, by an independent reviewer |

### 29.3 Honest trade-offs

1. **Unit cost.** Java Card inlays cost more than DESFire inlays. You sell each bracelet at ₦5,000 and recover it slowly through the instalment plan, so a higher unit cost raises your exposure (Section 28.3). Get written quotes at pilot and at 1,000-unit quantities before you fix the price.
2. **Schedule.** You cannot start the pilot until the applet exists, has been tested and has been reviewed. Development cards with known default keys are for experiments only, never for riders.
3. **The applet is the riskiest part of the project.** It holds the private key and does the signing. A bug there cannot be patched on bracelets already in riders' hands unless you built in a safe update path. Options for D16: write it in-house with an experienced Java Card reviewer, hire a contractor, or ask the chip vendor for a reference applet or platform you can adapt. Writing your own cryptographic applet from scratch is the highest-risk choice.
4. **Offline does not remove the need for a network.** A terminal can check offline that a bracelet is genuine. But the fare still has to be collected through Moniepoint, and Section 5.8 says only a succeeded payment makes a fare successful. An offline tap that is approved and charged later is the "float" behavior decision D2 is about. For the MVP, treat taps as online-only (D17) and use the Java Card for its security and its short, fast exchange, not for offline approval.
5. **Management keys still exist.** The private key on each bracelet is never on your server. But loading and updating applets uses GlobalPlatform management keys. Those must be protected too, and the card should be locked after provisioning so no one else can load applets.
6. **Different antenna rules.** Dual-interface secure elements have their own antenna requirements, such as load capacitance and coil inductance. Read the chip's datasheet before you finish the coil design, and tune with the NanoVNA once the real chip is in the shell.

### 29.4 Design

**Card side (applet).** At minimum:
- Select the applet
- Return the card's certificate
- Sign a challenge together with a counter, using the card's private key
- Increase the counter on every signature, so a replayed signature is detectable
- Never export the private key, and never accept a command that would let one be read

Keep the applet small and the command set minimal. Every extra command is more surface to review.

**Provisioning.**
1. Authenticate to the blank card with its management keys
2. Load the applet
3. The applet generates the key pair on the chip
4. Read the public key out
5. Your certificate authority signs a certificate for that public key
6. Write the certificate to the card
7. Record the public key, certificate serial, chip model and applet version in the database
8. Lock the card against further loading
9. Register the credential as INACTIVE and ship it INACTIVE

**Key custody.**
- CA root key: kept offline, used rarely, with two-person control
- Issuing (intermediate) key: in an HSM or cloud KMS, used by `cmd/provision` to sign certificates
- Management keys: derived per card from a management master in KMS, kept separate from the CA keys
- None of these ever sit in the repository, in logs, or on a terminal

**Tap protocol (server verifies, one network request per tap).**
1. While the ready screen is showing, the terminal asks `POST /tap/bracelet/challenge` and keeps a fresh single-use challenge ready. Doing this before the tap removes a network round trip from the moment the rider is waiting.
2. On tap, the terminal talks to the chip locally: select the applet, read the certificate, send the challenge and terminal ID, and receive the signature and counter. This is three or so short exchanges.
3. The terminal sends `POST /tap/bracelet/verify` with the certificate (or credential ID), the challenge ID, the signature and the counter.
4. The server then:
   - checks the certificate chain back to your CA
   - finds the credential and checks its state is ACTIVE
   - verifies the signature over challenge, terminal ID and counter
   - checks the counter is higher than the last one it recorded, and stores the new one in the same transaction
   - marks the challenge as used
   - passes the verified credential to `tap_service`, which does the rest as before

The challenge is short-lived and issued by the server, so a recorded tap cannot be replayed later. The counter adds a second check against cloning and replay.

**Why this removes the relay-latency worry.** Before, a DESFire tap needed several network trips, each carrying a chip message. Now the only network call at tap time is `verify`, plus a challenge that was fetched ahead of time.

### 29.5 Backend and schema changes

- `credential`: add `public_key bytea`, `cert_serial text`, `last_counter bigint`. Keep `chip_uid` only if you decide to store it, and never return it to clients.
- `bracelet`: remove `tier`. Add `chip_model`, `applet_version`, `provisioned_at`. Keep `edition` for cosmetic editions.
- `credential` type values: `bracelet_javacard` for all bracelets. Keep `phone_hce` for later. `bracelet_desfire` stays reserved but unused for now.
- New table for challenges: `tap_challenge (id, terminal_id, value, expires_at, used_at)`. Mark `used_at` in the same transaction as verification so a challenge cannot succeed twice.
- Authenticator: the Java Card implementation can be a single call, for example `Verify(ctx, TapProof) (credentialID, error)`. The multi-step session shape in 7A.3 is no longer needed for the main path. Keep it only if a later credential type needs it.
- Endpoints: replace `/tap/bracelet/start`, `/step`, `/finish` with `/tap/bracelet/challenge` and `/tap/bracelet/verify`.
- Revocation: state lives on the server, so a BLOCKED credential is declined at `verify`. For any offline check later, terminals also hold a blocklist with sequence numbers.
- Audit events: applet loaded, key generated, certificate issued, card locked, certificate revoked.

### 29.6 Revised phases

| Phase | Work | Gate |
|---|---|---|
| 0. Now, no hardware | Everything in 21.3, plus the schema in 29.5 and the challenge and verify endpoints against a fake chip | Fake bracelets tap, block and replay correctly in tests |
| 1. Development kit | Java Card development cards, a PC/SC reader, the Java Card development kit and GlobalPlatformPro. Write a tiny applet that signs a challenge. Verify the signature in Go | Signature produced on a development card verifies on the server |
| 2. Applet and CA | Full applet with counter. Certificate authority with HSM or KMS signing. `cmd/provision` writes certificates | A blank card becomes a registered, locked bracelet in one run |
| 3. Terminal | Android app does challenge, local chip exchange and verify. Measure end-to-end tap time | Under about one second on a good connection, clear message on a bad one |
| 4. Shell | Check the chip's antenna requirements, print with dummy inlay, tune and measure, then reprint with the real chip | Reliable reads on a wrist at the angles drivers see |
| 5. Review | Independent review of the applet, provisioning, CA custody and server verification | Written findings addressed. This is a gate, not an optional step |
| 6. Pilot | 10 to 20 bracelets with the instalment ceiling from Section 28 | A full operating day cash-free, no unexplained charge |

Phase 1 can run in parallel with Phase 0 once you have a reader and development cards. If vendor onboarding takes weeks, keep Phase 0 moving and treat chip access as the thing to start chasing now.

### 29.7 Threat model changes

Update Section 22.1:

- **Clone a bracelet:** now requires extracting a private key from a secure element. Much harder than copying data, but not impossible for a well-funded attacker. Keep per-rider limits.
- **Master key leak:** the old "all bracelet keys derive from one master key" risk is gone for tap verification. The new crown jewels are the CA signing keys and the management keys.
- **CA compromise:** an attacker could mint certificates for fake bracelets. Defense: HSM or KMS for the issuing key, an offline root, two-person control, and server-side registration so a valid certificate alone is not enough (the credential must also exist in your database and be ACTIVE).
- **Applet bug:** the new risk with the largest consequence. Defense: minimal commands, independent review, and a planned update path.
- **Supply chain:** buy chips only from authorized distributors, check each batch, and keep a record of chip serials.
- **Rogue reader:** a generic reader can still select the applet and ask for a signature, but it cannot get a valid challenge from your server. Test this.

### 29.8 Decisions

| ID | Decision | Status |
|---|---|---|
| D14 | Single tier, Java Card in every bracelet | Decided |
| D15 | Chip family and vendor (for example NXP JCOP or Infineon SECORA families), with quotes at pilot and scale quantities, and confirmation of contactless ECC support and antenna requirements | Open. Start the vendor conversations now |
| D16 | Who writes and maintains the applet: in-house with expert review, contractor, or vendor-supplied | Open |
| D17 | Offline taps | Online-only for the MVP. Revisit with D2 |
| D18 | Default limits policy by rider history, not product tier | Open |

Remove tier-based rows from any earlier table in the guide. Add these rows to Section 20.

### 29.9 Changes to other sections

| Section | Change |
|---|---|
| Bracelet guide, tier table | Superseded. One tier |
| 7A.1, 7A.2, 7A.3, 7A.4 | Replaced by 29.4 and 29.5 |
| 7A.5 risks | Replace the DESFire library and Random ID rows with applet, CA custody and vendor access |
| 14.5 rate limits | Limits now apply to `challenge` and `verify`. `challenge` is the one an attacker would flood |
| 21.2 to 21.6 | Replaced by 29.6 and the provisioning flow in 29.4 |
| 22 | Update as in 29.7 |
| 24.2 terminal app | Prefetch a challenge while ready. Do chip exchange locally. One verify call per tap |
| 25 website | No tier page. Show colorways and editions. Do not describe the security more strongly than the review supports |
| 26 | Java Card moves from "later" to core study (29.10) |
| 28 | Unit cost and exposure use the Java Card cost |

### 29.10 Knowledge to add

- **Java Card platform:** Oracle's Java Card specifications and development kit. Learn applets, APDU handling, transactions and atomicity, memory limits.
- **GlobalPlatform:** card lifecycle (loading, locking), secure channels, management keys. Use GlobalPlatformPro to experiment.
- **ECDSA on P-256:** key generation, signing, verification in Go (the standard library covers it).
- **Certificates:** X.509 or a compact certificate format that fits the card's memory, certificate chains, revocation.
- **HSM and KMS signing:** how to sign certificates without ever exporting the key, access control, audit.
- **Secure applet practices:** keep the command set tiny, validate every input length, avoid timing differences, test failure paths such as power loss mid-write. Ask your reviewer for their checklist before you start.
- **Vendor documentation:** chip datasheets, antenna design notes and the vendor's developer program, once you have NDA access.
- **Books and references:** *Smart Card Handbook* (Rankl and Effing) and *Serious Cryptography* (Aumasson) from Section 26 still apply. **Verify** a current Java Card textbook with the vendor or reviewer, since some older ones predate current versions.

### 29.11 Build order

1. Start the vendor and distributor conversations (D15). This has the longest wait
2. Decide D16, D17, D18
3. Schema and endpoints from 29.5 against a fake chip
4. Development cards, reader and the tiny signing applet (Phase 1)
5. Go verification of a real chip's signature
6. CA design with HSM or KMS signing, then `cmd/provision`
7. Full applet with counter, then the terminal app
8. Shell, coil and antenna work with the real chip
9. Independent review
10. Pilot
