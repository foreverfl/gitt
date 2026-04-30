---
sidebar_position: 2
---

# Create a Document

Documents can create **groups of pages** through:

- **Sidebar**
- **Previous/next navigation**
- **Versioning**

## Create Your First Doc

Create a file at `docs/hello.md`:

```md
# Hello

This is my **first Docusaurus document**!
```

A new document is now available at `http://localhost:3000/docs/hello`.

## Configure the Sidebar

Docusaurus automatically **creates a sidebar** from the `docs` folder.

You can customize the sidebar order and labels by adding metadata:

```md
---
sidebar_label: 'Hi!'
sidebar_position: 3
---

# Hello

This is my **first Docusaurus document**!
```

You can also create an explicit sidebar in `sidebars.js`:

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