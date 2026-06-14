---
title: "Quick start"
description: "Fetch your first record with open-elevation."
weight: 30
---

Once `open-elevation` is on your `PATH`, fetch a page. The argument is the path
of the page on open-elevation.com (everything after the host), or a full URL:

```bash
open-elevation page <path>
```

By default you get an aligned table. Ask for JSON when you want to pipe it:

```bash
$ open-elevation page <path> -o json
[
  {
    "id": "<path>",
    "url": "https://open-elevation.com/<path>",
    "title": "<path>",
    "body": "..."
  }
]
```

## Shape the output

The same flags work on every command:

```bash
open-elevation page <path> --fields id,url        # keep only these columns
open-elevation page <path> --template '{{.Body}}' # just the body text
open-elevation page <path> -o jsonl | jq .url     # one object per line, into jq
```

`-o` takes `table`, `json`, `jsonl`, `csv`, `tsv`, `url`, or `raw`. Left to
`auto`, it prints a table to a terminal and JSONL into a pipe, so the same
command reads well by hand and parses cleanly downstream. See
[output formats](/reference/output/) for the full contract.

## Follow the links

`links` lists the pages a page links to, and each one is a path you can fetch in
turn:

```bash
open-elevation links <path> -n 10                 # the first ten links
open-elevation links <path> -o url                # just the URLs
open-elevation links <path> -o url | head -3 | xargs -n1 open-elevation page
```

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
open-elevation serve --addr :7777 &
curl -s 'localhost:7777/v1/page/<path>'          # NDJSON, one record per line
open-elevation mcp                                # MCP over stdio: page, links
```

## What to build next

This scaffold ships one example type, `page`, wired end to end so the whole
chain works today. To make it really about open-elevation, model the records you
care about in `open-elevation/` and declare their operations in
`open-elevation/domain.go`. Each one you add shows up as a command here, a route
under `serve`, and a tool under `mcp`, with no extra wiring. The
[guides](/guides/) cover the common jobs.
