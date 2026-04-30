---
sidebar_position: 1
---

# 페이지 만들기

Docusaurus에서 새로운 **페이지**를 만드는 방법을 배워봅시다.

## 새 페이지 추가하기

`src/pages` 디렉토리에 React 컴포넌트를 추가하여 새 페이지를 만들 수 있습니다.

- `src/pages/index.js` → `localhost:3000/`
- `src/pages/foo.md` → `localhost:3000/foo`
- `src/pages/foo/bar.js` → `localhost:3000/foo/bar`

## 첫 번째 React 페이지 만들기

`src/pages/my-react-page.js` 파일을 생성하세요:

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

이제 `http://localhost:3000/my-react-page`에서 새 페이지에 접근할 수 있습니다.

## 첫 번째 Markdown 페이지 만들기

`src/pages/my-markdown-page.md` 파일을 생성하세요:

```markdown
# My Markdown page

This is a Markdown page
```

이제 `http://localhost:3000/my-markdown-page`에서 새 페이지에 접근할 수 있습니다.