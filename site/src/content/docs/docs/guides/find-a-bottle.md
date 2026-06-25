---
title: Find a bottle
description: The full workflow — search the catalog, get the product code, and check live stock near you.
sidebar:
  order: 1
---

The core loop: **search → code → inventory**.

## 1. Search the catalog

`product search` queries Virginia ABC's live web catalog, so it finds everything the website
does — including new and online-only bottles that the downloadable price list omits.

```bash
vabc product search "old forester"
vabc product search oftd --select productCode,name,size,retailPrice
```

Filter by category/type or allocated status:

```bash
vabc product search rye --type bourbon
vabc product search "" --allocated         # browse allocated releases
```

Every row carries a 6-digit **`productCode`** — that's the key for everything else.

## 2. Check who has it near you

`inventory check` takes a product code plus a location. `--near` accepts a **ZIP**, a **street
address**, or **`lat,lng`** — distances are measured from that point.

```bash
vabc inventory check 953714 --near 22182
vabc inventory check 953714 --near "1100 Bank St, Richmond VA"
vabc inventory check 953714 --store 219      # anchor on a specific store
```

You get the anchor store's quantity plus nearby stores that stock it, ranked by distance. For
statewide availability:

```bash
vabc inventory warehouse 953714
```

## 3. Find stores

```bash
vabc store near 22182 --limit 5
vabc store get 219
vabc store list
```

## 4. Track allocated / lottery releases

```bash
vabc lottery check 953714
```

Returns active limited-availability events for a product. Event titles are CMS-authored free
text and are **fenced as untrusted** by default.

## Scripting it

```bash
# Lowest price among in-stock nearby stores, as JSON
vabc --json inventory check 010807 --near 22182 \
  | jq '[.store, .nearbyStores[]] | map(select(.quantity > 0)) | sort_by(.distance)[0]'
```
