import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'getting-started',
    'setting-up-agent',
    {
      type: 'category',
      label: 'Features',
      link: {
        type: 'doc',
        id: 'features/index',
      },
      items: [
        'features/live-agents',
        'features/context-system',
        'features/agent-configuration',
        'features/agent-templates',
        'features/skills',
        'features/plugins',
        'features/multi-ai',
        'features/crontask',
        'features/git-auto-sync',
        'features/task-stop',
        'features/concurrency-quota',
        'features/persistence',
        'features/enterprise',
        'features/pod-configuration',
      ],
    },
    {
      type: 'category',
      label: 'Use Cases',
      link: {
        type: 'doc',
        id: 'use-cases/index',
      },
      items: [
        'use-cases/agent-plugins',
        'use-cases/docker-in-docker',
        'use-cases/vscode-in-browser',
      ],
    },
    'architecture',
    'security',
    {
      type: 'category',
      label: 'Operations',
      items: [
        'operations/troubleshooting',
        'operations/releasing',
        'operations/upgrading',
      ],
    },
    'roadmap',
  ],
};

export default sidebars;
