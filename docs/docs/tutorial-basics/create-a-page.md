---
sidebar_position: 1
---

# Create a Page

Learn how to create new **pages** in Docusaurus.

## Add a New Page

You can create a new page by adding React components to the `src/pages` directory.

- `src/pages/index.js` → `localhost:3000/`
- `src/pages/foo.md` → `localhost:3000/foo`
- `src/pages/foo/bar.js` → `localhost:3000/foo/bar`

## Create Your First React Page

Create a file at `src/pages/my-react-page.js`:

```jsx
import React from 'react';
import Layout from '@theme/Layout';

export default function MyReactPage() {
  return (
    <Layout>
      <h1>My React page</h1>
      <p>This is a React page</p>
    </Layout>
  );
}
```

Now you can access the new page at `http://localhost:3000/my-react-page`.

## Create Your First Markdown Page

Create a file at `src/pages/my-markdown-page.md`:

```markdown
# My Markdown page

This is a Markdown page
```

Now you can access the new page at `http://localhost:3000/my-markdown-page`.