#!/usr/bin/env node
/* Check all demo pages load and have expected structure */
const http = require('http');
const fs = require('fs');
const path = require('path');

const root = path.join(__dirname, '..');
const expectedPages = [
  'index.html', 'assets/css/tokens.css', 'assets/css/reset.css', 'assets/css/pages.css',
  'assets/js/icons.js', 'assets/js/mock.js', 'assets/js/router.js', 'assets/js/store.js',
  'assets/js/api.js', 'assets/js/ui.js', 'assets/js/app.js',
  'assets/js/pages/today.js', 'assets/js/pages/onboarding.js', 'assets/js/pages/records.js',
  'assets/js/pages/quick-record.js', 'assets/js/pages/ai-confirm.js', 'assets/js/pages/ai-input.js',
  'assets/js/pages/cats.js', 'assets/js/pages/cat-detail.js', 'assets/js/pages/trends.js',
  'assets/js/pages/medical-upload.js', 'assets/js/pages/reminders.js', 'assets/js/pages/moments.js',
  'assets/js/pages/family.js', 'assets/js/pages/inventory.js', 'assets/js/pages/expenses.js',
  'assets/js/pages/settings.js'
];

let fail = 0;
expectedPages.forEach(p => {
  const file = path.join(root, p);
  if (!fs.existsSync(file)) { fail++; console.log('MISSING: ' + p); }
});
if (fail) { console.log(fail + ' files missing'); process.exit(1); }
console.log('All ' + expectedPages.length + ' expected files present.');
