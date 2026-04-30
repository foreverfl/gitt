import React from 'react';
import Layout from '@theme/Layout';
import Translate, { translate } from '@docusaurus/Translate';
import Link from '@docusaurus/Link';
import versions from '@site/versions.json';

interface VersionItemProps {
  version: string;
  isCurrentVersion?: boolean;
  isNextVersion?: boolean;
}

function VersionItem({ version, isCurrentVersion, isNextVersion }: VersionItemProps): React.ReactElement {
  const versionName = isNextVersion
    ? translate({
        id: 'versionsPage.versionEntry.next',
        message: 'Next',
        description: 'The label for the next unreleased version'
      })
    : version;

  const versionLabel = isCurrentVersion
    ? translate({
        id: 'versionsPage.versionEntry.stable',
        message: 'Stable',
        description: 'The label for stable versions'
      })
    : isNextVersion
    ? translate({
        id: 'versionsPage.versionEntry.canary',
        message: 'Canary 🚧',
        description: 'The label for canary/unreleased versions'
      })
    : '';

  const docsUrl = isNextVersion
    ? '/docs/next/intro'
    : isCurrentVersion
    ? '/docs/intro'
    : `/docs/${version}/intro`;

  const releaseNotesUrl = `https://github.com/foreverfl/kb-base/releases/tag/v${version}`;

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-6 hover:border-green-500 dark:hover:border-green-500 transition-colors duration-200 bg-white dark:bg-gray-800">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="flex items-center gap-3">
          <h3 className="text-xl font-bold text-gray-900 dark:text-white m-0">
            {versionName}
          </h3>
          {versionLabel && (
            <span className={`px-3 py-1 text-sm font-semibold rounded-full ${
              isCurrentVersion
                ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                : 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
            }`}>
              {versionLabel}
            </span>
          )}
        </div>
        <div className="flex gap-3">
          <Link
            to={docsUrl}
            className="inline-flex items-center px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors duration-200 no-underline hover:no-underline"
          >
            <Translate
              id="versionsPage.versionEntry.documentation"
              description="The label for documentation link">
              Documentation
            </Translate>
          </Link>
          {!isNextVersion && (
            <a
              href={releaseNotesUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors duration-200 no-underline hover:no-underline"
            >
              <Translate
                id="versionsPage.versionEntry.releaseNotes"
                description="The label for release notes link">
                Release Notes
              </Translate>
            </a>
          )}
        </div>
      </div>
    </div>
  );
}

export default function Versions(): React.ReactElement {
  const latestVersion = versions[0];
  const pastVersions = versions.slice(1);

  return (
    <Layout
      title={translate({
        id: 'versionsPage.title',
        message: 'Versions',
        description: 'The title of the versions page'
      })}
      description={translate({
        id: 'versionsPage.description',
        message: 'Documentation versions page',
        description: 'The description of the versions page'
      })}
    >
      <main className="container mx-auto px-4 py-12 max-w-5xl">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white mb-4">
            <Translate
              id="versionsPage.header.title"
              description="The main title of the versions page">
              Documentation Versions
            </Translate>
          </h1>
          <p className="text-lg text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
            <Translate
              id="versionsPage.header.description"
              description="The description below the main title">
              Here you can find all available versions of the documentation.
            </Translate>
          </p>
        </div>

        {/* Current Version */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
            <Translate
              id="versionsPage.currentVersion.title"
              description="The title for current version section">
              Current Version (Stable)
            </Translate>
          </h2>
          <p className="text-gray-600 dark:text-gray-400 mb-4">
            <Translate
              id="versionsPage.currentVersion.description"
              description="The description for current version section">
              This is the latest stable version of the documentation.
            </Translate>
          </p>
          <VersionItem version={latestVersion} isCurrentVersion />
        </section>

        {/* Next Version */}
        <section className="mb-12">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
            <Translate
              id="versionsPage.nextVersion.title"
              description="The title for next version section">
              Next Version (Unreleased)
            </Translate>
          </h2>
          <p className="text-gray-600 dark:text-gray-400 mb-4">
            <Translate
              id="versionsPage.nextVersion.description"
              description="The description for next version section">
              This is the documentation for the unreleased version currently in development.
            </Translate>
          </p>
          <VersionItem version="next" isNextVersion />
        </section>

        {/* Past Versions */}
        {pastVersions.length > 0 && (
          <section>
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
              <Translate
                id="versionsPage.pastVersions.title"
                description="The title for past versions section">
                Past Versions
              </Translate>
            </h2>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              <Translate
                id="versionsPage.pastVersions.description"
                description="The description for past versions section">
                Here you can find documentation for previous versions.
              </Translate>
            </p>
            <div className="space-y-4">
              {pastVersions.map((version: string) => (
                <VersionItem key={version} version={version} />
              ))}
            </div>
          </section>
        )}
      </main>
    </Layout>
  );
}