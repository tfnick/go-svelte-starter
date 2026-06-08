export type OrderStatus = 'pending' | 'paid' | 'shipped' | 'completed' | 'cancelled';

type EnumOption<T extends string> = {
  value: T;
  label: string;
};

const orderStatusDefinitions: EnumOption<OrderStatus>[] = [
  { value: 'pending', label: '待支付' },
  { value: 'paid', label: '已支付' },
  { value: 'shipped', label: '已发货' },
  { value: 'completed', label: '已完成' },
  { value: 'cancelled', label: '已取消' }
];

export const orderStatusOptions = orderStatusDefinitions.map((option) => ({ ...option }));

const orderStatusLabels = new Map<OrderStatus, string>(
  orderStatusDefinitions.map((option) => [option.value, option.label])
);

export function orderStatusLabel(value: string): string {
  return orderStatusLabels.get(value as OrderStatus) || value;
}
