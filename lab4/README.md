# Lab 4 — Coffee Varka (SSG + Git CMS)

Static landing page for a fictional coffee shop, migrated from the plain
HTML/CSS Lab 3 onto a Static Site Generator and a Git-based headless CMS.

**Live:** https://coffee-varka.netlify.app
**Admin (CMS):** https://coffee-varka.netlify.app/admin/

## Stack

- **SSG:** [Eleventy (11ty) v3](https://www.11ty.dev/) — Nunjucks templates, markdown collections, JSON data files.
- **CMS:** [Decap CMS](https://decapcms.org/) (formerly Netlify CMS) authenticated via [Netlify Identity](https://docs.netlify.com/security/secure-access-to-sites/identity/) and committing to GitHub through [Git Gateway](https://docs.netlify.com/security/secure-access-to-sites/git-gateway/).
- **CSS:** [Tailwind CSS](https://tailwindcss.com/) (CDN build) with a custom palette (`coffee-*`), Google Fonts (Playfair Display + Lato) and hand-written keyframe animations.
- **Hosting:** [Netlify](https://www.netlify.com/) (auto-deploy on push to `main`).

## Project layout

```
lab4/
├── .eleventy.js              # Eleventy config (input/output, sorted collections)
├── netlify.toml              # build command + publish dir
├── package.json
└── src/
    ├── _data/site.json       # ALL editable site copy (hero, about, contact, theme, …)
    ├── _includes/layouts/
    │   └── base.njk          # shared HTML shell (head, header, footer, mascot)
    ├── admin/
    │   ├── index.html        # Decap CMS entry point
    │   └── config.yml        # collections + Site Settings field schema
    ├── assets/
    │   ├── images/           # logo + uploaded media (CMS writes here)
    │   └── reset.css
    ├── menu/                 # markdown entries (one per menu item)
    ├── testimonials/         # markdown entries
    ├── gallery/              # markdown entries
    └── index.njk             # the homepage template
```

## Editable content

Almost every visible string and image is editable via the CMS at `/admin/`:

| Where it shows | What's editable |
|---|---|
| `<head>` | site title, meta description, author, year |
| Header | logo image, nav link list (label + href), accent theme color |
| Hero | two-line heading, subheading, two CTA buttons (label + link) |
| About | heading, two paragraphs, image URL + alt text |
| Menu | full collection (add/edit/delete items: title, description, price, image, order) + section heading & subheading |
| Testimonials | full collection (author, role, quote, order) + section heading |
| Gallery | full collection (alt, image, "wide" flag, order) + section heading & subheading |
| Contact | heading, subheading, address, hours, email, phone, form button label |
| Mascot | toggle on/off, tooltip title + message |
| Mobile CTAs | three independently toggleable banners (Quick Order, Download App, sticky bar) — heading, subheading, button label & link |
| Footer | social URLs (Instagram, Facebook, TikTok), year |

Saving in the CMS triggers a commit to `main` → Netlify rebuilds and the
change is live in ~30 seconds.

## Run locally

```bash
cd lab4
npm install
npm start            # serves with live reload at http://localhost:8080
# or
npm run build        # one-shot build into _site/
```

The CMS UI (`/admin/`) only works against the deployed site, because Git
Gateway needs a Netlify Identity session. Locally you preview content by
editing the markdown / JSON files directly.

## Deploy

Pushing to `main` on [github.com/dmracovit/PWeb](https://github.com/dmracovit/PWeb) automatically triggers a
Netlify build (base directory `lab4`, command `npm run build`, publish
`_site`). The CMS itself also pushes to `main` whenever an editor saves,
so the deploy loop is the same in both directions.
