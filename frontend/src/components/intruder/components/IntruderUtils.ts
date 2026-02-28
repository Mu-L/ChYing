// 定义有效载荷相关的工具函数
import type { PayloadPosition, IntruderResult, AttackResult, IntruderTab } from '../../../types/intruder';
import { i18n } from '@/i18n';

export function useIntruderUtils() {
  // 统一的颜色定义
  const colors = [
    { id: 'default', value: '#4f46e5', labelKey: 'modules.intruder.default_color' },
    { id: 'red', value: '#ef4444', labelKey: 'modules.intruder.red' },
    { id: 'green', value: '#10b981', labelKey: 'modules.intruder.green' },
    { id: 'blue', value: '#3b82f6', labelKey: 'modules.intruder.blue' },
    { id: 'yellow', value: '#f59e0b', labelKey: 'modules.intruder.yellow' },
    { id: 'orange', value: '#f97316', labelKey: 'modules.intruder.orange' },
    { id: 'teal', value: '#14b8a6', labelKey: 'modules.intruder.teal' },
  ];

  // 获取本地化颜色选项
  const getLocalizedColorOptions = () => {
    return colors.map(c => ({
      id: c.id,
      value: c.value,
      label: i18n.global.t(c.labelKey)
    }));
  };

  // 保留向后兼容的别名
  const getEnglishColorOptions = () => getLocalizedColorOptions();
  const getChineseColorOptions = () => getLocalizedColorOptions();

  // 为整个请求添加有效载荷标记
  const wrapSelectionInRequestWithPayloadMarker = (
    request: string,
    selectedText: string
  ): string => {
    if (!request.includes(selectedText)) return request;
    
    // 将选中文本用标记符号包围
    return request.replace(selectedText, `§${selectedText}§`);
  };

  // 清除整个请求中的所有有效载荷标记
  const clearAllPayloadMarkersInRequest = (request: string): string => {
    // 替换所有§§标记
    return request.replace(/§/g, '');
  };

  // 提取payload位置的函数
  const extractPayloadPositions = (text: string, payloadMarker = '$'): PayloadPosition[] => {
    const positions: PayloadPosition[] = [];
    let index = text.indexOf(payloadMarker);
    let positionIndex = 0;
    
    while (index !== -1) {
      const endIndex = text.indexOf(payloadMarker, index + payloadMarker.length);
      if (endIndex === -1) break;
      
      // 获取标记之间的文本
      const value = text.substring(index + payloadMarker.length, endIndex);
      
      // 尝试确定这是否是参数
      let paramName: string | undefined;
      const beforeMarker = text.substring(0, index).trim();
      const lastLineBreak = beforeMarker.lastIndexOf('\n');
      const lastLine = lastLineBreak !== -1 ? beforeMarker.substring(lastLineBreak + 1) : beforeMarker;
      
      if (lastLine.includes(':')) {
        // 可能是头部
        const headerName = lastLine.split(':')[0].trim();
        paramName = headerName;
      } else if (lastLine.includes('=')) {
        // 可能是查询参数或表单数据
        const paramParts = lastLine.split('=');
        paramName = paramParts[0].trim();
      }
      
      positions.push({
        start: index,
        end: endIndex + payloadMarker.length,
        value,
        paramName,
        index: positionIndex  // 添加 index 属性
      });
      
      positionIndex++;  // 增加位置索引
      index = text.indexOf(payloadMarker, endIndex + payloadMarker.length);
    }
    
    return positions;
  };

  // 格式化入侵者结果
  const formatIntruderResult = (result: any): IntruderResult => {
    return {
      id: Number(result.id || 0),
      payload: Array.isArray(result.payload) ? result.payload : [String(result.payload || '')],
      status: result.status,
      length: result.length,
      timeMs: result.timeMs,
      timestamp: String(result.timestamp || ''),
      request: result.request,
      response: result.response,
      color: result.color,
      selected: result.selected
    };
  };

  // 统一的错误处理函数
  const handleError = (error: unknown, context: string): void => {
    console.error(`入侵者模块错误 (${context}):`, error);
    // 可以添加更多处理，如通知系统等
  };

  // 确保载荷集类型兼容性
  const ensurePayloadSetTypeCompatibility = (payloadSets: any[]): any[] => {
    if (!payloadSets || !Array.isArray(payloadSets)) return [];
    
    return payloadSets.map(set => {
      const newSet = { ...set };
      // 将 'brute-forcer' 转换为 'brute-force'，确保类型兼容
      if (newSet.type === 'brute-forcer') {
        newSet.type = 'brute-force';
      }
      return newSet;
    });
  };
  
  // IntruderResult 转 AttackResult
  const intruderResultToAttackResult = (result: IntruderResult): AttackResult => {
    return {
      id: String(result.id),
      payload: result.payload,
      status: result.status,
      length: result.length,
      timeMs: result.timeMs,
      request: result.request,
      response: result.response,
      timestamp: typeof result.timestamp === 'string' ? new Date(result.timestamp).getTime() : Date.now()
    };
  };
  
  // AttackResult 转 IntruderResult
  const attackResultToIntruderResult = (result: AttackResult): IntruderResult => {
    return {
      id: Number(result.id || 0),
      payload: result.payload,
      status: result.status,
      length: result.length,
      timeMs: result.timeMs,
      timestamp: typeof result.timestamp === 'number' 
        ? new Date(result.timestamp).toISOString() 
        : new Date().toISOString(),
      request: result.request,
      response: result.response
    };
  };
  
  // 高阶函数处理错误
  const withErrorHandling = <T extends (...args: any[]) => any>(fn: T, context: string) => {
    return (...args: Parameters<T>): ReturnType<T> | undefined => {
      try {
        return fn(...args);
      } catch (error) {
        handleError(error, context);
        return undefined;
      }
    };
  };

  return {
    // 颜色相关
    colors,
    getEnglishColorOptions,
    getChineseColorOptions,
    getLocalizedColorOptions,
    
    // 载荷标记相关
    wrapSelectionInRequestWithPayloadMarker,
    clearAllPayloadMarkersInRequest,
    extractPayloadPositions,
    
    // 结果处理相关
    formatIntruderResult,
    handleError,
    ensurePayloadSetTypeCompatibility,
    
    // 新增方法
    intruderResultToAttackResult,
    attackResultToIntruderResult,
    withErrorHandling
  };
}