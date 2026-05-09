import React, { useEffect, useState } from 'react';
import {
  Button,
  Form,
  Header,
  Label,
  Modal,
  Pagination,
  Segment,
  Select,
  Table,
} from 'semantic-ui-react';
import {
  API,
  copy,
  isAdmin,
  showError,
  showSuccess,
  showWarning,
  timestamp2string,
} from '../helpers';
import { useTranslation } from 'react-i18next';

import { ITEMS_PER_PAGE } from '../constants';
import { renderColorLabel, renderQuota } from '../helpers/render';
import { Link } from 'react-router-dom';

const PAGE_SIZE_OPTIONS_V2 = [
  { key: 10, text: '10', value: 10 },
  { key: 25, text: '25', value: 25 },
  { key: 50, text: '50', value: 50 },
  { key: 100, text: '100', value: 100 },
];

function renderTimestamp(timestamp, request_id) {
  return (
    <code
      onClick={async () => {
        if (await copy(request_id)) {
          showSuccess(`已复制请求 ID：${request_id}`);
        } else {
          showWarning(`请求 ID 复制失败：${request_id}`);
        }
      }}
      style={{ cursor: 'pointer' }}
    >
      {timestamp2string(timestamp)}
    </code>
  );
}

const MODE_OPTIONS = [
  { key: 'all', text: '全部用户', value: 'all' },
  { key: 'self', text: '当前用户', value: 'self' },
];

function renderType(type) {
  switch (type) {
    case 1:
      return (
        <Label basic color='green'>
          充值
        </Label>
      );
    case 2:
      return (
        <Label basic color='olive'>
          消费
        </Label>
      );
    case 3:
      return (
        <Label basic color='orange'>
          管理
        </Label>
      );
    case 4:
      return (
        <Label basic color='purple'>
          系统
        </Label>
      );
    case 5:
      return (
        <Label basic color='violet'>
          测试
        </Label>
      );
    default:
      return (
        <Label basic color='black'>
          未知
        </Label>
      );
  }
}

function getColorByElapsedTime(elapsedTime) {
  if (elapsedTime === undefined || 0) return 'black';
  if (elapsedTime < 1000) return 'green';
  if (elapsedTime < 3000) return 'olive';
  if (elapsedTime < 5000) return 'yellow';
  if (elapsedTime < 10000) return 'orange';
  return 'red';
}

function renderDetail(log) {
  const detailText = typeof log.content === 'string' ? log.content : '';
  return (
    <>
      <div
        style={{
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          maxWidth: '100%',
        }}
      >
        {detailText}
      </div>
      <br />
      {log.elapsed_time && (
        <Label
          basic
          size={'mini'}
          color={getColorByElapsedTime(log.elapsed_time)}
        >
          {log.elapsed_time} ms
        </Label>
      )}
      {log.is_stream && (
        <>
          <Label size={'mini'} color='pink'>
            Stream
          </Label>
        </>
      )}
      {log.system_prompt_reset && (
        <>
          <Label basic size={'mini'} color='red'>
            System Prompt Reset
          </Label>
        </>
      )}
    </>
  );
}

const renderTextLogString = (log) => {
  let typeStr = log.type === 1 ? '充值' : log.type === 2 ? '消费' : log.type === 3 ? '管理' : log.type === 4 ? '系统' : '其他';
  let timeStr = timestamp2string(log.created_at);
  let isWorker = log.content && log.content.includes('Edge Proxy') ? 'Worker: YES' : 'Worker: NO';
  let isStreamStr = log.is_stream ? 'Stream: YES' : 'Stream: NO';
  let latencyStr = log.elapsed_time ? (log.elapsed_time / 1000).toFixed(2) + 's' : '0.00s';
  let speed = (log.elapsed_time && log.completion_tokens) ? (log.completion_tokens / (log.elapsed_time / 1000)).toFixed(2) : '0.00';
  
  let str = `[${timeStr}] [${typeStr}] [${isWorker}] [${isStreamStr}]\n`;
  str += `[路由] ReqID: ${log.request_id || '-'} | Channel: #${log.channel || 0} (KeyIdx: #${log.channel_key_index || 0})\n`;
  str += `[会话] User: ${log.username || '-'} | AuthToken: ${log.token_name || '-'}\n`;
  str += `[模型] ${log.model_name || '-'}\n`;
  str += `[用量] Prompt: ${log.prompt_tokens || 0} | Completion: ${log.completion_tokens || 0} | Total: ${(log.prompt_tokens || 0) + (log.completion_tokens || 0)} (Quota: ${log.quota || 0})\n`;
  str += `[性能] 总耗时: ${latencyStr} | 平均出字速率: ${speed} t/s\n`;
  if (log.system_prompt_reset) {
    str += `[附加] System Prompt Reset: YES\n`;
  }
  str += `[详情] ${log.content || 'N/A'}\n`;
  return str;
};

const LogsTable = () => {
  const { t } = useTranslation();
  const [logs, setLogs] = useState([]);
  const [selectedLog, setSelectedLog] = useState(null);
  const [showStat, setShowStat] = useState(false);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searching, setSearching] = useState(false);
  const [logType, setLogType] = useState(0);
  const [viewMode, setViewMode] = useState('table');
  const [errorOnly, setErrorOnly] = useState(false);
  const isAdminUser = isAdmin();
  let now = new Date();
  const [inputs, setInputs] = useState({
    username: '',
    token_name: '',
    model_name: '',
    start_timestamp: timestamp2string(0),
    end_timestamp: timestamp2string(now.getTime() / 1000 + 3600),
    channel: '',
  });
  const {
    username,
    token_name,
    model_name,
    start_timestamp,
    end_timestamp,
    channel,
  } = inputs;

  const [stat, setStat] = useState({
    quota: 0,
    token: 0,
  });

  const LOG_OPTIONS = [
    { key: '0', text: t('log.type.all'), value: 0 },
    { key: '1', text: t('log.type.topup'), value: 1 },
    { key: '2', text: t('log.type.usage'), value: 2 },
    { key: '3', text: t('log.type.admin'), value: 3 },
    { key: '4', text: t('log.type.system'), value: 4 },
    { key: '5', text: t('log.type.test'), value: 5 },
  ];

  const handleInputChange = (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const getLogSelfStat = async () => {
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let res = await API.get(
      `/api/log/self/stat?type=${logType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`
    );
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const getLogStat = async () => {
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let res = await API.get(
      `/api/log/stat?type=${logType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}`
    );
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const handleEyeClick = async () => {
    if (!showStat) {
      if (isAdminUser) {
        await getLogStat();
      } else {
        await getLogSelfStat();
      }
    }
    setShowStat(!showStat);
  };

  const showUserTokenQuota = () => {
    return logType !== 5;
  };

  const loadLogs = async (startIdx) => {
    let url = '';
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    if (isAdminUser) {
      url = `/api/log/?p=${startIdx}&page_size=${pageSize}&type=${logType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}`;
    } else {
      url = `/api/log/self/?p=${startIdx}&page_size=${pageSize}&type=${logType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    }
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      if (startIdx === 0) {
        setLogs(data);
      } else {
        let newLogs = [...logs];
        newLogs.splice(startIdx * pageSize, data.length, ...data);
        setLogs(newLogs);
      }
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const onPaginationChange = (e, { activePage }) => {
    (async () => {
      if (activePage === Math.ceil(logs.length / pageSize) + 1) {
        // In this case we have to load more data and then append them.
        await loadLogs(activePage - 1);
      }
      setActivePage(activePage);
    })();
  };

  const refresh = async () => {
    setLoading(true);
    setActivePage(1);
    await loadLogs(0);
  };

  useEffect(() => {
    refresh().then();
  }, [logType, pageSize]);

  const getVisibleLogsV2 = () => {
    return logs.filter((log) => {
      if (log.deleted) return false;
      if (errorOnly) {
        return typeof log.content === 'string' && (log.content.includes('错误') || log.content.toLowerCase().includes('error') || log.type === 4);
      }
      return true;
    });
  };

  const getCurrentPageLogsV2 = () => {
    const visibleLogs = getVisibleLogsV2();
    return visibleLogs.slice(
      (activePage - 1) * pageSize,
      activePage * pageSize
    );
  };

  const getTotalPagesV2 = () => {
    return Math.ceil(logs.length / pageSize) + (logs.length % pageSize === 0 ? 1 : 0);
  };

  const handlePageSizeChangeV2 = (e, { value }) => {
    setPageSize(value);
    setActivePage(1);
  };

  const searchLogs = async () => {
    if (searchKeyword === '') {
      // if keyword is blank, load files instead.
      await loadLogs(0);
      setActivePage(1);
      return;
    }
    setSearching(true);
    const res = await API.get(`/api/log/self/search?keyword=${searchKeyword}`);
    const { success, message, data } = res.data;
    if (success) {
      setLogs(data);
      setActivePage(1);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  const handleKeywordChange = async (e, { value }) => {
    setSearchKeyword(value.trim());
  };

  const sortLog = (key) => {
    if (logs.length === 0) return;
    setLoading(true);
    let sortedLogs = [...logs];
    if (typeof sortedLogs[0][key] === 'string') {
      sortedLogs.sort((a, b) => {
        return ('' + a[key]).localeCompare(b[key]);
      });
    } else {
      sortedLogs.sort((a, b) => {
        if (a[key] === b[key]) return 0;
        if (a[key] > b[key]) return -1;
        if (a[key] < b[key]) return 1;
      });
    }
    if (sortedLogs[0].id === logs[0].id) {
      sortedLogs.reverse();
    }
    setLogs(sortedLogs);
    setLoading(false);
  };

  const openLogTextModal = (log) => {
    setSelectedLog(log);
  };

  const closeLogTextModal = () => {
    setSelectedLog(null);
  };

  const copyLogText = async (log) => {
    const detailText = typeof log?.content === 'string' ? log.content : '';
    if (!detailText) {
      showWarning(t('log.messages.empty_detail'));
      return;
    }
    if (await copy(detailText)) {
      showSuccess(t('log.messages.copy_success'));
    } else {
      showWarning(t('log.messages.copy_failed'));
    }
  };

  return (
    <>
      <Header as='h3'>
        {t('log.usage_details')}（{t('log.total_quota')}：
        {showStat && renderQuota(stat.quota, t)}
        {!showStat && (
          <span
            onClick={handleEyeClick}
            style={{ cursor: 'pointer', color: 'gray' }}
          >
            {t('log.click_to_view')}
          </span>
        )}
        ）
      </Header>
      <Form>
        <Form.Group>
          <Form.Input
            fluid
            label={t('log.table.token_name')}
            size={'small'}
            width={3}
            value={token_name}
            placeholder={t('log.table.token_name_placeholder')}
            name='token_name'
            onChange={handleInputChange}
          />
          <Form.Input
            fluid
            label={t('log.table.model_name')}
            size={'small'}
            width={3}
            value={model_name}
            placeholder={t('log.table.model_name_placeholder')}
            name='model_name'
            onChange={handleInputChange}
          />
          <Form.Input
            fluid
            label={t('log.table.start_time')}
            size={'small'}
            width={4}
            value={start_timestamp}
            type='datetime-local'
            name='start_timestamp'
            onChange={handleInputChange}
          />
          <Form.Input
            fluid
            label={t('log.table.end_time')}
            size={'small'}
            width={4}
            value={end_timestamp}
            type='datetime-local'
            name='end_timestamp'
            onChange={handleInputChange}
          />
          <Form.Button
            fluid
            label={t('log.buttons.query')}
            size={'small'}
            width={2}
            onClick={refresh}
          >
            {t('log.buttons.submit')}
          </Form.Button>
        </Form.Group>
        {isAdminUser && (
          <>
            <Form.Group>
              <Form.Input
                fluid
                label={t('log.table.channel_id')}
                size={'small'}
                width={3}
                value={channel}
                placeholder={t('log.table.channel_id_placeholder')}
                name='channel'
                onChange={handleInputChange}
              />
              <Form.Input
                fluid
                label={t('log.table.username')}
                size={'small'}
                width={3}
                value={username}
                placeholder={t('log.table.username_placeholder')}
                name='username'
                onChange={handleInputChange}
              />
            </Form.Group>
          </>
        )}
        <Form.Group inline>
          <Form.Field>
            <Form.Input
              icon='search'
              placeholder={t('log.search')}
              value={searchKeyword}
              onChange={(e, { value }) => setSearchKeyword(value)}
            />
          </Form.Field>
          <Form.Checkbox
            label='仅看错误 (Error Only)'
            checked={errorOnly}
            onChange={(e, { checked }) => {
              setActivePage(1);
              setErrorOnly(checked);
            }}
          />
          <Form.Checkbox
            label='纯文本日志 (Text Mode)'
            checked={viewMode === 'text'}
            onChange={(e, { checked }) => setViewMode(checked ? 'text' : 'table')}
          />
          <Form.Field>
            <label>{t('log.table.page_size')}</label>
            <Select
              compact
              options={PAGE_SIZE_OPTIONS_V2}
              value={pageSize}
              onChange={handlePageSizeChangeV2}
            />
          </Form.Field>
          {viewMode === 'text' && (
            <Button size='mini' basic onClick={() => {
              const textOutput = getVisibleLogsV2()
              .map(log => renderTextLogString(log))
              .join('==================================================\n');
              copy(textOutput).then(() => showSuccess('已复制纯文本（Copied!）'));
            }}>
              一键复制所有加载文本 (Copy All loaded)
            </Button>
          )}
        </Form.Group>
      </Form>
      {viewMode === 'table' ? (
      <Table basic={'very'} compact size='small'>
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('created_time');
              }}
              width={3}
            >
              {t('log.table.time')}
            </Table.HeaderCell>
            {isAdminUser && (
              <Table.HeaderCell
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  sortLog('channel');
                }}
                width={1}
              >
                {t('log.table.channel')}
              </Table.HeaderCell>
            )}
            {isAdminUser && (
              <Table.HeaderCell
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  sortLog('channel_key_index');
                }}
                width={1}
              >
                {t('log.table.channel_key')}
              </Table.HeaderCell>
            )}
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('type');
              }}
              width={1}
            >
              {t('log.table.type')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('model_name');
              }}
              width={2}
            >
              {t('log.table.model')}
            </Table.HeaderCell>
            {showUserTokenQuota() && (
              <>
                {isAdminUser && (
                  <Table.HeaderCell
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                      sortLog('username');
                    }}
                    width={2}
                  >
                    {t('log.table.username')}
                  </Table.HeaderCell>
                )}
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('token_name');
                  }}
                  width={2}
                >
                  {t('log.table.token_name')}
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('prompt_tokens');
                  }}
                  width={1}
                >
                  {t('log.table.prompt_tokens')}
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('completion_tokens');
                  }}
                  width={1}
                >
                  {t('log.table.completion_tokens')}
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('quota');
                  }}
                  width={1}
                >
                  {t('log.table.quota')}
                </Table.HeaderCell>
              </>
            )}
            <Table.HeaderCell>{t('log.table.detail')}</Table.HeaderCell>
          </Table.Row>
        </Table.Header>

        <Table.Body>
          {getCurrentPageLogsV2().map((log, idx) => {
              return (
                <Table.Row key={log.id}>
                  <Table.Cell>
                    {renderTimestamp(log.created_at, log.request_id)}
                  </Table.Cell>
                  {isAdminUser && (
                    <Table.Cell>
                      {log.channel ? (
                        <Label
                          basic
                          as={Link}
                          to={`/channel/edit/${log.channel}`}
                        >
                          {log.channel}
                        </Label>
                      ) : (
                        ''
                      )}
                    </Table.Cell>
                  )}
                  {isAdminUser && (
                    <Table.Cell>
                      {log.channel_key_index ? (
                        <Label basic>{`#${log.channel_key_index}`}</Label>
                      ) : log.channel_key_id ? (
                        <Label basic>{log.channel_key_id}</Label>
                      ) : (
                        ''
                      )}
                    </Table.Cell>
                  )}
                  <Table.Cell>{renderType(log.type)}</Table.Cell>
                  <Table.Cell>
                    {log.model_name ? renderColorLabel(log.model_name) : ''}
                  </Table.Cell>
                  {showUserTokenQuota() && (
                    <>
                      {isAdminUser && (
                        <Table.Cell>
                          {log.username ? (
                            <Label
                              basic
                              as={Link}
                              to={`/user/edit/${log.user_id}`}
                            >
                              {log.username}
                            </Label>
                          ) : (
                            ''
                          )}
                        </Table.Cell>
                      )}
                      <Table.Cell>
                        {log.token_name ? renderColorLabel(log.token_name) : ''}
                      </Table.Cell>

                      <Table.Cell>
                        {log.prompt_tokens ? log.prompt_tokens : ''}
                      </Table.Cell>
                      <Table.Cell>
                        {log.completion_tokens ? log.completion_tokens : ''}
                      </Table.Cell>
                      <Table.Cell>
                        {log.quota ? renderQuota(log.quota, t, 6) : ''}
                      </Table.Cell>
                    </>
                  )}

                  <Table.Cell>
                    {renderDetail(log)}

                  </Table.Cell>
                </Table.Row>
              );
            })}
        </Table.Body>

        <Table.Footer>
          <Table.Row>
            <Table.HeaderCell colSpan={'10'}>
              <Select
                placeholder={t('log.type.select')}
                options={LOG_OPTIONS}
                style={{ marginRight: '8px' }}
                name='logType'
                value={logType}
                onChange={(e, { name, value }) => {
                  setLogType(value);
                }}
              />
              <Button size='small' onClick={refresh} loading={loading}>
                {t('log.buttons.refresh')}
              </Button>
              <span style={{ marginLeft: '12px', marginRight: '8px' }}>
                {t('log.table.page_size')}
              </span>
              <Select
                compact
                options={PAGE_SIZE_OPTIONS_V2}
                value={pageSize}
                onChange={handlePageSizeChangeV2}
              />
              <Pagination
                floated='right'
                activePage={activePage}
                onPageChange={onPaginationChange}
                size='small'
                siblingRange={1}
                totalPages={getTotalPagesV2()}
              />
            </Table.HeaderCell>
          </Table.Row>
        </Table.Footer>
      </Table>
      ) : (
        <Segment basic>
          <pre style={{
            fontFamily: 'monospace',
            whiteSpace: 'pre-wrap',
            wordWrap: 'break-word',
            background: '#f4f4f4',
            padding: '16px',
            borderRadius: '6px'
          }}>
            {getCurrentPageLogsV2()
              .map(log => renderTextLogString(log))
              .join('==================================================\n')}
          </pre>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '14px' }}>
            <div>
              <Select
                placeholder={t('log.type.select')}
                options={LOG_OPTIONS}
                style={{ marginRight: '8px' }}
                name='logType'
                value={logType}
                onChange={(e, { name, value }) => {
                  setLogType(value);
                }}
              />
              <Button size='small' onClick={refresh} loading={loading}>
                {t('log.buttons.refresh')}
              </Button>
              <span style={{ marginLeft: '12px', marginRight: '8px' }}>
                {t('log.table.page_size')}
              </span>
              <Select
                compact
                options={PAGE_SIZE_OPTIONS_V2}
                value={pageSize}
                onChange={handlePageSizeChangeV2}
              />
            </div>
            <Pagination
              activePage={activePage}
              onPageChange={onPaginationChange}
              size='small'
              siblingRange={1}
              totalPages={getTotalPagesV2()}
            />
          </div>
        </Segment>
      )}
      <Modal open={!!selectedLog} onClose={closeLogTextModal} size='large'>
        <Modal.Header>{t('log.modal.title')}</Modal.Header>
        <Modal.Content scrolling>
          <pre
            style={{
              margin: 0,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              fontFamily: 'monospace',
              background: '#f8f8f8',
              padding: '12px',
              borderRadius: '6px',
            }}
          >
            {selectedLog?.content || ''}
          </pre>
        </Modal.Content>
        <Modal.Actions>
          <Button basic onClick={() => copyLogText(selectedLog)}>
            {t('log.buttons.copy_text')}
          </Button>
          <Button onClick={closeLogTextModal}>{t('log.buttons.close')}</Button>
        </Modal.Actions>
      </Modal>
    </>
  );
};

export default LogsTable;
