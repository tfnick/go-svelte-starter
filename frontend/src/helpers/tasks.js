export const taskTypes = Object.freeze({
  ordersExcelExport: 'orders_excel_export'
});

export function taskTitle(task) {
  if (task?.task_type === taskTypes.ordersExcelExport) return 'Orders Excel export';
  return task?.task_type || 'Task';
}

export function taskResult(task) {
  try {
    return JSON.parse(task?.result_json || '{}') || {};
  } catch {
    return {};
  }
}

export function taskFilename(task) {
  return taskResult(task).filename || 'orders.xlsx';
}

export function canDownloadTask(task) {
  return task?.status === 'completed' && task?.task_type === taskTypes.ordersExcelExport;
}
