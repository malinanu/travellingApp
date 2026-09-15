export * from './types';
export * from './api';
export * from './upload';
export * from './use-auto-save';
export * from './vendor-categories';
export * from './supported-icons';

export function formatCurrency(amount: number, currency = 'LKR'): string {
  if (currency === 'LKR' || currency === 'Rs' || currency === 'Rs.') {
    return `Rs. ${amount.toLocaleString('en-LK')}`;
  }
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(amount);
}

