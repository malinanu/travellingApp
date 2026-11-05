'use client';

// Packages & Pricing Management — Spotto Command Center
// Interactive Kanban / Drag-and-Drop builder to configure vendor subscription packages
// ($3.99, $11.99, $22.99) and master feature allocations.

import Shell from '../../components/shell';
import PackageBuilder from '../../components/package-builder';

export default function PackagesPage() {
  return (
    <Shell>
      <PackageBuilder />
    </Shell>
  );
}
