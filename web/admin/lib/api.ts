'use client';

import { createApi } from '@spotto/shared';

/** Command-center API client — token separate from the vendor portal's. */
export const api = createApi('spotto_admin_token');
