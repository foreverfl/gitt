---
sidebar_position: 2
---

# 문서 만들기

문서는 다음을 통해 **페이지 그룹**을 만들 수 있습니다:

- **사이드바**
- **이전/다음 탐색**
- **버전 관리**

## 첫 번째 문서 만들기

`docs/hello.md` 파일을 생성하세요:

```md
# Hello

This is my **first Docusaurus document**!
```

이제 `http://localhost:3000/docs/hello`에서 새 문서를 사용할 수 있습니다.

## 사이드바 구성

Docusaurus는 `docs` 폴더에서 자동으로 **사이드바를 생성**합니다.

메타데이터를 추가하여 사이드바 순서와 레이블을 사용자 정의할 수 있습니다:

```md
---
sidebar_label: 'Hi!'
sidebar_position: 3
---

# Hello

This is my **first Docusaurus document**!
```

`sidebars.js`에서 명시적인 사이드바를 만들 수도 있습니다:

```js
module.exports = {
  tutorialSidebar: [
    'intro',
    // highlight-next-line
    'hello',
    {
      type: 'category',
      label: 'Tutorial',
      items: ['tutorial-basics/create-a-document'],
    },
  ],
};
```