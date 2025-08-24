import { assertEquals } from '@std/assert';
import { LogFileNameFactory } from './LogFileName.ts';

class MockDateConstructor {
  constructor(value?: string | Date) {
    if (value === undefined) {
      return new Date('2022-08-30');
    }
    return new Date(value);
  }
}
const factory = new LogFileNameFactory(MockDateConstructor as DateConstructor);

Deno.test('target value is undefined', () => {
  const actual = factory.create();

  assertEquals(actual.name, '2022-08-30');
});

Deno.test('target value is typeof DateString', () => {
  const actual = factory.create('2022-07-31');

  assertEquals(actual.name, '2022-07-31');
});

Deno.test('target value is typeof Date', () => {
  const targetDate = new Date('2022-04-04');
  const actual = factory.create(targetDate);

  assertEquals(actual.name, '2022-04-04');
});
