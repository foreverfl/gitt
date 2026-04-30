---
sidebar_position: 1
---

# チュートリアルの紹介

**5分以内にDocusaurusを学びましょう**。

## はじめに

**新しいサイトを作成**して始めましょう。

または、**[docusaurus.new](https://docusaurus.new)**で**Docusaurusをすぐに試す**こともできます。

### 必要なもの

- [Node.js](https://nodejs.org/en/download/) バージョン18.0以上:
  - Node.jsをインストールする際は、依存関係に関連するすべてのチェックボックスを選択することをお勧めします。

## 新しいサイトを生成

**クラシックテンプレート**を使用して新しいDocusaurusサイトを生成します。

コマンドを実行すると、クラシックテンプレートが自動的にプロジェクトに追加されます:

```bash
npm init docusaurus@latest my-website classic
```

このコマンドは、コマンドプロンプト、Powershell、ターミナル、またはコードエディタの統合ターミナルに入力できます。

このコマンドは、Docusaurusを実行するために必要なすべての依存関係もインストールします。

## サイトを起動

開発サーバーを実行します:

```bash
cd my-website
npm run start
```

`cd`コマンドは、作業中のディレクトリを変更します。新しく作成したDocusaurusサイトで作業するには、ターミナルでそこに移動する必要があります。

`npm run start`コマンドは、Webサイトをローカルでビルドし、開発サーバーを介して提供します。http://localhost:3000/で確認できます。

`docs/intro.md`(このページ)を開いて数行編集してみてください: サイトが**自動的にリロード**され、変更が表示されます。