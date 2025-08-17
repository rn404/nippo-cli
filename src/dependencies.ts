export { join } from '@std/path';
export { parse } from '@std/flags';
export { ensureDir, ensureFile } from '@std/fs';
// Using Node.js compatible crypto for hash functions
import { createHash } from 'node:crypto';
export { createHash };

import { walk } from '@std/fs';

// Replaced with native APIs
const homeDir = (): string | undefined => {
  return Deno.env.get('HOME') || Deno.env.get('USERPROFILE');
};

// Replaced with Intl.DateTimeFormat
const formatTime = (date: Date): string => {
  return new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).format(date);
};

const formatDate = (date: Date): string => {
  return new Intl.DateTimeFormat('en-CA').format(date);
};

export { homeDir, walk, formatTime, formatDate };

export {
  Command,
  HelpCommand,
} from 'cliffy';
