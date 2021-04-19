import omit from 'lodash/omit'

export type RecordSet<Parent extends string, Child extends string, Value> =
  Record<Parent, Record<Child, Value>>

export function setChild<
  Parent extends string,
  Child extends string,
  Value,
> (
  obj: Record<Parent, Record<Child, Value>>,
  parentKey: Parent,
  childKey: Child,
  value: Value,
): Record<Parent, Record<Child, Value>> {
  const inner = obj[parentKey] || {}

  return {
    ...obj,
    [parentKey]: {
      ...inner,
      [childKey]: value,
    },
  }
}

export function removeChild<
  Parent extends string,
  Child extends string,
  Value,
> (
  obj: RecordSet<Parent, Child, Value>,
  parentKey: Parent,
  childKey: Child,
): Record<Parent, Record<Child, Value>> {
  let inner = (obj[parentKey] || {}) as Record<Child, Value>
  // I don't want to fight with you TypeScript, but sometimes you make my life
  // damn hard.
  inner = omit(inner, childKey) as Record<Child, Value>


  if (Object.keys(inner).length === 0) {
    return removeParent(obj, parentKey)
  }

  return {
    ...obj,
    [parentKey]: inner,
  }
}

export function removeParent<
  Parent extends string,
  Child extends string,
  Value,
> (
  obj: RecordSet<Parent, Child, Value>,
  parentKey: Parent,
): Record<Parent, Record<Child, Value>> {
  return omit(obj, parentKey) as unknown as RecordSet<Parent, Child, Value>
}
