import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import Heading from '@theme/Heading';
import CodeBlock from '@theme/CodeBlock';

import styles from './index.module.css';

const agentYaml = `apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: dev-agent
spec:
  profile: "Interactive development agent"
  workspaceDir: /workspace
  port: 4096
  persistence:
    sessions:
      size: "2Gi"
  credentials:
    - name: api-key
      secretRef:
        name: ai-credentials
        key: api-key
      env: OPENCODE_API_KEY`;

const exampleYaml = `apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: update-dependencies
spec:
  agentRef:
    name: dev-agent
  description: |
    Update all dependencies to latest versions.
    Run tests and create a pull request.`;

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <p className={styles.heroDescription}>
          Run AI coding agents as live services on Kubernetes. Deploy persistent agents
          your team can interact with anytime &mdash; or run batch tasks at scale.
          Built on OpenCode, designed for teams and enterprise.
        </p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/getting-started">
            Get Started
          </Link>
          <Link
            className="button button--outline button--lg"
            style={{color: 'white', borderColor: 'white', marginLeft: '1rem'}}
            href="https://github.com/kubeopencode/kubeopencode">
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

function QuickExample() {
  return (
    <section className={styles.quickExample}>
      <div className="container">
        <div className="row">
          <div className="col col--6">
            <Heading as="h2">Deploy a Live Agent in Minutes</Heading>
            <p>
              Run AI coding agents as persistent services on Kubernetes.
              Your team can interact with them anytime &mdash; through the
              web terminal, CLI, or by submitting Tasks.
            </p>
            <ul>
              <li>Zero cold start &mdash; agent is always running</li>
              <li>Interactive terminal access via CLI or web</li>
              <li>Shared context across all tasks</li>
              <li>Session history persists across restarts</li>
            </ul>
          </div>
          <div className="col col--6">
            <CodeBlock language="yaml" title="agent.yaml">
              {agentYaml}
            </CodeBlock>
          </div>
        </div>
        <div className="row" style={{marginTop: '2rem'}}>
          <div className="col col--6">
            <Heading as="h2">Submit Tasks as YAML</Heading>
            <p>
              Define what you want done as a Task. Submit it to a persistent Agent
              for interactive work, or use an AgentTemplate for ephemeral one-off tasks.
            </p>
            <ul>
              <li>No new tools to learn &mdash; just <code>kubectl apply</code></li>
              <li>Works with any CI/CD pipeline</li>
              <li>Scale with Helm templates for batch operations</li>
              <li>Monitor with standard Kubernetes tooling</li>
            </ul>
          </div>
          <div className="col col--6">
            <CodeBlock language="yaml" title="task.yaml">
              {exampleYaml}
            </CodeBlock>
          </div>
        </div>
      </div>
    </section>
  );
}

export default function Home(): ReactNode {
  return (
    <Layout
      title="Kubernetes-native Agent Platform for Teams and Enterprise"
      description="Deploy, manage, and govern AI coding agents at scale on Kubernetes. Built on OpenCode, designed for teams and enterprise.">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
        <QuickExample />
      </main>
    </Layout>
  );
}
