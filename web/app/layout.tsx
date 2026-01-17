import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'IssueSight',
  description: 'Bridging the gap between "Good First Issues" and "Great First Contributions"',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
