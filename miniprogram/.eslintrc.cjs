module.exports = {
  root: true,
  env: {
    es2022: true,
    node: true
  },
  extends: ['plugin:vue/vue3-essential'],
  parser: 'vue-eslint-parser',
  parserOptions: {
    parser: '@typescript-eslint/parser',
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  plugins: ['@typescript-eslint'],
  globals: {
    uni: 'readonly',
    getCurrentPages: 'readonly'
  },
  ignorePatterns: ['dist/', 'node_modules/', 'src/components/app/iconData.ts'],
  rules: {
    'vue/multi-word-component-names': 'off'
  }
}
