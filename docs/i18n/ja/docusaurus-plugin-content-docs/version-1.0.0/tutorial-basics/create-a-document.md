---
sidebar_position: 2
---

# ドキュメントの作成

ドキュメントは次の方法で**ページのグループ**を作成できます:

- **サイドバー**
- **前/次のナビゲーション**
- **バージョン管理**

## 最初のドキュメントを作成

`docs/hello.md`ファイルを作成します:

```md
# Hello

This is my **first Docusaurus document**!
```

`http://localhost:3000/docs/hello`で新しいドキュメントが利用可能になります。

## サイドバーの設定

Docusaurusは`docs`フォルダから自動的に**サイドバーを作成**します。

メタデータを追加することで、サイドバーの順序とラベルをカスタマイズできます:

```md
---
sidebar_label: 'Hi!'
sidebar_position: 3
---

# Hello

This is my **first Docusaurus document**!
```

`sidebars.js`で明示的なサイドバーを作成することもできます:

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