import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { isPromoterVerifyCodeValid } from './phoneRegisterTask.js'

const __dirname = dirname(fileURLToPath(import.meta.url))
const componentSource = readFileSync(
  resolve(__dirname, '../components/PhoneRegisterTaskCenter.vue'),
  'utf8'
)

assert.equal(isPromoterVerifyCodeValid('123456'), true)
assert.equal(isPromoterVerifyCodeValid('12345'), false)
assert.equal(isPromoterVerifyCodeValid('1234567'), false)
assert.equal(isPromoterVerifyCodeValid('12a456'), false)
assert.equal(isPromoterVerifyCodeValid(' 123456 '), true)

assert.equal(componentSource.includes('maxlength="6"'), false)
assert.equal(componentSource.includes('@input="handleVerifyCodeInput'), false)
