import {test} from 'node:test'
import assert from 'node:assert/strict'
import {tuvCharge} from '../src/tuvCharge.ts'

test('agreed layered model, including corrected examples',()=>{
  for(const [upper,lower,expected] of [[45,35,0],[50,35,13.5],[57,35,38],[60,35,42.6],[60,45,55.3],[60,50,75.5],[60,57,100]]){
    assert.equal(tuvCharge(upper,lower),expected,`${upper}/${lower}`)
  }
})
test('empty, full and unusable upper layer',()=>{
  assert.equal(tuvCharge(0,0),0)
  assert.equal(tuvCharge(45,60),0)
  assert.equal(tuvCharge(57,57),100)
  assert.equal(tuvCharge(100,100),100)
  assert.equal(tuvCharge(51,51),50)
})
test('invalid or unavailable temperature never becomes zero',()=>{
  for(const value of [null,NaN,Infinity,-Infinity,-1,101]){
    assert.equal(tuvCharge(value,60),null)
    assert.equal(tuvCharge(60,value),null)
  }
})
test('warming either sensor cannot reduce the estimate',()=>{
  for(const upper of [46,50,57,60,80]){
    let previous=0
    for(let lower=0;lower<=100;lower++){
      const value=tuvCharge(upper,lower)!
      assert.ok(value>=previous&&value>=0&&value<=100)
      previous=value
    }
  }
  for(const lower of [0,35,45,50,57]){
    let previous=0
    for(let upper=0;upper<=100;upper++){
      const value=tuvCharge(upper,lower)!
      assert.ok(value>=previous&&value>=0&&value<=100)
      previous=value
    }
  }
})
