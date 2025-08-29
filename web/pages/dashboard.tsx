import { GetServerSideProps } from 'next';
import fs from 'fs';
import path from 'path';
import styles from '../styles/Dashboard.module.css';

interface DashboardProps {
  dashboardHtml: string;
}

export default function Dashboard({ dashboardHtml }: DashboardProps) {
  return (
    <div
      className={styles.dashboardContainer}
      dangerouslySetInnerHTML={{ __html: dashboardHtml }}
    />
  );
}

export const getServerSideProps: GetServerSideProps = async () => {
  try {
    const dashboardPath = path.join(process.cwd(), 'dashboard.html');
    const dashboardHtml = fs.readFileSync(dashboardPath, 'utf8');

    return {
      props: {
        dashboardHtml,
      },
    };
  } catch (error) {
    console.error('Error loading dashboard:', error);
    return {
      props: {
        dashboardHtml: '<h1>Error loading dashboard</h1>',
      },
    };
  }
};
