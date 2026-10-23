# Image input through the two CLIs (plan 31, 2026-10-23)

How each agent CLI takes an image with the owner's message through the
interface the dashboard already drives, and what the model behind it
accepts. Every claim names its source and the date it was read.

## Claude Code: stream-json

The dashboard runs `claude -p --input-format stream-json --output-format
stream-json` and writes one user message a line,
`{"type":"user","message":{"role":"user","content":<text>},"parent_tool_use_id":null,"session_id":"default"}`
(`internal/provider/claude/session.go`, `userMessage`). The Agent SDK's
streaming-input page documents the same message with `content` as the
Messages API's content blocks, an image among them
(https://code.claude.com/docs/en/agent-sdk/streaming-vs-single-mode, read
2026-10-23):

```json
{"type": "user", "message": {"role": "user", "content": [
  {"type": "text", "text": "Review this architecture diagram"},
  {"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": "<base64>"}}
]}, "parent_tool_use_id": null}
```

The page lists "Image uploads: attach images directly to messages" among
streaming input's capabilities and says single-message input does not
support "direct image attachments in messages". It also says: "If an image
block's source is missing or isn't an object, the SDK doesn't report an
error. Claude Code sends Claude a text note in place of the image, such as
`[Image could not be processed: image block has no source object]`, and
the session continues." The SDK's TypeScript and Python packages wrap the
CLI's stream-json, so the CLI's input is the SDK's message.

The model's limits (https://platform.claude.com/docs/en/build-with-claude/vision,
read 2026-10-23):
- Formats: JPEG, PNG, GIF and WebP (`image/jpeg`, `image/png`, `image/gif`,
  `image/webp`). "Animations are unsupported, and only the first frame is
  used."
- Size: "10 MB (base64-encoded) when using the Claude API directly"; the
  request 32 MB; "The maximum dimensions for each image are 8000x8000 px";
  over 20 images in one request, 2000 px a side.
- Count: "100 for each request on the API, for models with a 200k-token
  context window", 600 otherwise.
- Placement: "Claude works best when images come before text."
- Metadata: "Claude does not parse or receive any metadata from images
  passed to it", so an EXIF orientation is not applied by the model.
- Cost: patches of 28x28 px, `⌈width / 28⌉ × ⌈height / 28⌉` visual tokens;
  large images are downscaled (1,568 px a side on the standard tier,
  2,576 px on the high-resolution tier).

## Codex: the app-server

The dashboard drives `codex app-server` over JSON-RPC and starts a turn
with `turn/start` (`internal/provider/codex/session.go`, `Send`), whose
`input` is a list of `UserInput`. The protocol's schema, probed in plan 1
(`doc/research/probes/p0-codex-app-server-schema-v2.json`, `UserInput`),
has three members:

- `{"type": "text", "text": …}`;
- `{"type": "image", "url": …, "detail": …?}` (`ImageUserInput`), the URL
  a web URL or a data URL;
- `{"type": "localImage", "path": …, "detail": …?}` (`LocalImageUserInput`),
  a file the CLI reads itself.

The model's limits (https://developers.openai.com/api/docs/guides/images-vision,
read 2026-10-23): PNG, JPEG, WebP and non-animated GIF; "up to 1,500
images per request" and "up to 512 MB total payload per request"; each
image "up to 30,000 patches" of 32x32 px after resizing, over which the
image is "rejected, not automatically resized to meet it"; the `detail`
parameter `low`, `high`, `original` or `auto`; "images may be resized
before analysis"; filenames and metadata are not processed.

## What follows for the dashboard

- Both CLIs take an image as content of the message, so the dashboard
  never writes a path into the text.
- The formats both models accept are PNG, JPEG, WebP and GIF (still). Every
  other format needs a conversion to PNG here, which needs a decoder the
  dashboard has: Go's standard library decodes PNG, JPEG and GIF; the
  dashboard adds its own readers for BMP, uncompressed baseline TIFF, ICO
  and the WebP header (plan 31, D1). HEIC, AVIF, JPEG XL, JPEG 2000, PSD,
  TGA, DDS, HDR, EXR and camera RAW have no decoder here and are refused by
  name; SVG is never rendered or executed.
- Animation: Claude uses the first frame; OpenAI refuses animated GIFs. The
  dashboard sends an animated GIF's first frame as PNG and says so; an
  animated WebP is refused, since taking its first frame needs a decoder.
- Orientation: neither model reads EXIF, so a JPEG whose EXIF orientation
  is not 1 is re-encoded with the rotation applied before it is sent.
- Limits applied when the image is attached: Claude Code 7.5 MB a file (10
  MB once base64-encoded), 8000 px a side, 100 images; Codex 20 MB a file
  (the dashboard's own cap under the 512 MB request), 30,000 patches of
  32 px, 100 images (the dashboard's own cap).

## Seen live (2026-10-23)

Plan 31's live tests (doc/plan31-evaluation.md, Follow-up), one turn on
each CLI with the generated PNG:
- Claude Code 2.1.303, on claude-haiku-5-5: the base64 image block before
  the text was read; the answer named the three colours and the text.
  Nothing of the image came back on its stdout (the dashboard passes no
  `--replay-user-messages`).
- Codex 0.163.0, on gpt-6-sol: an `image` input whose `url` is a data URL
  is accepted and read; the answer named the three colours and the text.
  The image's base64 came back on its stdout in two lines, the
  userMessage item's `item/started` and `item/completed`, at
  `params.item.content[0].url`: the `UserInput` sent (the third run, at
  01:47). So the dashboard removes image data URLs from every line of
  Codex's it keeps (plan 31).
- Both CLIs keep each message, images included, in their own session
  history: Claude Code's project transcript under `~/.claude/projects`,
  Codex's rollout under `~/.codex/sessions` (read after the run, each
  holding the image's base64). The dashboard cannot change that, and keeps
  no copy of its own.
