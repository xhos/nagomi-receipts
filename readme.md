# nagomi-receipts

Parses receipt photos into line items for [nagomi](https://github.com/xhos/nagomi), using a vision model through Ollama or Gemini. Stateless.

Locally, `qwen2.5vl:3b` is the smallest model that works. It needs a fair amount of RAM and times out on weak hardware. Gemini's free tier is plenty if you don't mind sending receipts to Google.

## config

See [.env.example](.env.example).

## dev

```bash
run
go run ./cmd/test-cli image.jpg   # parse one image
regen
bump-protos
```
