---
sidebar_position: 1
---

# ページの作成

Docusaurusで新しい**ページ**を作成する方法を学びましょう。

## 新しいページを追加

`src/pages`ディレクトリにReactコンポーネントを追加することで、新しいページを作成できます。

- `src/pages/index.js` → `localhost:3000/`
- `src/pages/foo.md` → `localhost:3000/foo`
- `src/pages/foo/bar.js` → `localhost:3000/foo/bar`

## 最初のReactページを作成

`src/pages/my-react-page.js`ファイルを作成します:

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

`http://localhost:3000/my-react-page`で新しいページにアクセスできます。

## 最初のMarkdownページを作成

`src/pages/my-markdown-page.md`ファイルを作成します:

```markdown
# My Markdown page

This is a Markdown page
```

`http://localhost:3000/my-markdown-page`で新しいページにアクセスできます。