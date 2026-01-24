import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Header } from '@/components/layout/Header';
import './globals.css';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
});

export const metadata: Metadata = {
  title: 'IssueSight - Transform GitHub Issues into Learning Tutorials',
  description: 'IssueSight analyzes GitHub issues and generates comprehensive, step-by-step tutorials. Turn complex problems into teachable moments.',
  keywords: ['GitHub', 'tutorials', 'AI', 'learning', 'open source'],
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={inter.variable}>
      <body>
        <Header />
        <main>{children}</main>
      </body>
    </html>
  );
}
