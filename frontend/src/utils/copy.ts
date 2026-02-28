// 复制功能
import { message } from "./message";
import { i18n } from '@/i18n';

export function copyList(data: string[]) {
    const apiList = data.join("\n"); // 将 API 列表转换为字符串
    navigator.clipboard.writeText(apiList).then(() => {
        message.success(i18n.global.t('utils.copy.copied'));
    }).catch(err => {
        message.error(i18n.global.t('utils.copy.copy_failed'));
        message.error(err);
        console.error(err);
    });
}

/**
 * 复制文本到剪贴板
 * @param text 要复制的文本
 * @returns Promise<void>
 */
export function copyToClipboard(text: string): Promise<void> { // 重命名并添加返回类型
  if (!navigator.clipboard) {
    console.warn('Clipboard API not available.');
    return Promise.reject(new Error('Clipboard API not available'));
  }
  return navigator.clipboard.writeText(text)
    .catch(err => {
      console.error('Failed to copy text to clipboard:', err);
      return Promise.reject(err); // 将错误继续抛出
    });
}