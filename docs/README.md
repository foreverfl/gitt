# Docusaurus + Tailwind CSS Starter Template (pnpm)

A clean starter template for building documentation sites with Docusaurus and Tailwind CSS. This template is optimized for pnpm package manager.

## Features

- **Docusaurus 3.x** - Fast and feature-rich static site generator
- **Tailwind CSS 4.x** - Utility-first CSS framework for rapid UI development
- **React 19** - Latest version of React for modern web development
- **Version Management** - Built-in versioning system with canary support
- **Dark mode support** - Built-in theme switching
- **Responsive design** - Mobile-friendly out of the box
- **SEO optimized** - Meta tags and structured data ready

## Quick Start

### Prerequisites

- Node.js 18.0 or higher
- pnpm package manager (recommended) or npm

### Installation

1. Clone this repository:
```bash
git clone <repository-url> my-site
cd my-site
```

2. Install dependencies:
```bash
pnpm install
# or using npm
npm install
```

3. Start the development server:
```bash
pnpm run start
# or using npm
npm run start
```

Your site will be available at `http://localhost:3000`

## Project Structure

```
my-site/
├── blog/
│   └── 2024-11-03-welcome.md   # Blog posts
├── docs/
│   └── intro.md                 # Documentation pages
├── src/
│   ├── components/              # React components
│   ├── css/                     # Global styles
│   └── pages/                   # Custom pages
├── static/                      # Static assets
├── docusaurus.config.js         # Docusaurus configuration
├── postcss.config.js           # PostCSS configuration
├── sidebars.js                 # Sidebar configuration
└── package.json
```

## Available Scripts

### Development
- `pnpm run start` - Start development server
- `pnpm run build` - Build for production
- `pnpm run serve` - Test production build locally
- `pnpm run clear` - Clear cache
- `pnpm run deploy` - Deploy to GitHub Pages (requires configuration)

### Versioning
- `pnpm run version <version>` - Create a new documentation version (e.g., `pnpm run version 1.1.0`)
- `pnpm run version:create <version>` - Alias for creating a new version
- `pnpm run version:list` - List all available versions

Note: You can also use `npm run` instead of `pnpm run` for these commands.

## Version Management

This project includes a built-in versioning system that allows you to maintain multiple versions of your documentation:

### Creating a New Version

To create a new version of your documentation:

```bash
pnpm run version 1.1.0
```

This will:
- Snapshot your current `docs/` folder into `versioned_docs/version-1.1.0/`
- Create a versioned sidebars file
- Update `versions.json` with the new version

### Version Structure

After creating versions, your project structure will include:

```
my-site/
├── versions.json                 # List of all versions
├── versioned_docs/
│   └── version-1.0.0/           # Snapshot of docs for version 1.0.0
└── versioned_sidebars/
    └── version-1.0.0-sidebars.json
```

### Current Version

The current version is displayed in:
- Navbar dropdown selector
- Homepage hero section with version badge
- Canary badge for bleeding-edge features

## Customization

### Site Configuration

Edit `docusaurus.config.js` to customize:
- Site metadata (title, tagline, URL)
- Navigation menu
- Footer links
- Theme settings

### Styling with Tailwind

- Use Tailwind utilities in React components
- Customize theme in `tailwind.config.js` (create if needed)
- Global styles in `src/css/custom.css`

### Adding Content

- **Documentation**: Add `.md` or `.mdx` files to the `docs/` directory
- **Blog posts**: Add dated markdown files to `blog/` directory
- **Custom pages**: Create React components in `src/pages/`

## Deployment

This template is ready for deployment to various platforms:

- GitHub Pages
- Vercel
- Netlify
- AWS Amplify

Refer to the [Docusaurus deployment documentation](https://docusaurus.io/docs/deployment) for specific instructions.

## License

This project is licensed under the ISC License.

## Resources

- [Docusaurus Documentation](https://docusaurus.io/)
- [Tailwind CSS Documentation](https://tailwindcss.com/)
- [React Documentation](https://react.dev/)