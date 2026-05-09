import React, {useEffect, useState} from 'react';
import {useTranslation} from 'react-i18next';
import {Button, Dropdown, Form, Input, Label, Message, Popup, Table} from 'semantic-ui-react';
import {Link} from 'react-router-dom';
import {
  API,
  loadChannelModels,
  setPromptShown,
  shouldShowPrompt,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
} from '../helpers';

import {CHANNEL_OPTIONS} from '../constants';
import {renderGroup, renderNumber} from '../helpers/render';

function renderTimestamp(timestamp) {
  return <>{timestamp2string(timestamp)}</>;
}

let type2label = undefined;

function renderType(type, t) {
  if (!type2label) {
    type2label = new Map();
    for (let i = 0; i < CHANNEL_OPTIONS.length; i++) {
      type2label[CHANNEL_OPTIONS[i].value] = CHANNEL_OPTIONS[i];
    }
    type2label[0] = {
      value: 0,
      text: t('channel.table.status_unknown'),
      color: 'grey',
    };
  }
  return (
    <Label basic color={type2label[type]?.color}>
      {type2label[type] ? type2label[type].text : type}
    </Label>
  );
}

function renderBalance(type, balance, t) {
  switch (type) {
    case 1:
      if (balance === 0) {
        return <span>{t('channel.table.balance_not_supported')}</span>;
      }
      return <span>${balance.toFixed(2)}</span>;
    case 4:
      return <span>楼{balance.toFixed(2)}</span>;
    case 8:
      return <span>${balance.toFixed(2)}</span>;
    case 5:
      return <span>楼{(balance / 10000).toFixed(2)}</span>;
    case 10:
      return <span>{renderNumber(balance)}</span>;
    case 12:
      return <span>楼{balance.toFixed(2)}</span>;
    case 13:
      return <span>{renderNumber(balance)}</span>;
    case 20:
      return <span>${balance.toFixed(2)}</span>;
    case 36:
      return <span>楼{balance.toFixed(2)}</span>;
    case 44:
      return <span>楼{balance.toFixed(2)}</span>;
    default:
      return <span>{t('channel.table.balance_not_supported')}</span>;
  }
}

function isShowDetail() {
  return localStorage.getItem('show_detail') === 'true';
}

const promptID = 'detail';

const ChannelsTable = () => {
  const { t } = useTranslation();
  const [channels, setChannels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searching, setSearching] = useState(false);
  const [updatingBalance, setUpdatingBalance] = useState(false);
  const [reorderingChannelAction, setReorderingChannelAction] = useState('');
  const [showPrompt, setShowPrompt] = useState(shouldShowPrompt(promptID));
  const [showDetail, setShowDetail] = useState(isShowDetail());

  const processChannelData = (channel) => {
    if (channel.models === '') {
      channel.models = [];
      channel.test_model = '';
    } else {
      channel.models = channel.models.split(',');
      if (channel.models.length > 0) {
        channel.test_model = channel.models[0];
      }
      channel.model_options = channel.models.map((model) => {
        return {
          key: model,
          text: model,
          value: model,
        };
      });
    }
    return channel;
  };

  const loadChannels = async (keyword = '') => {
    const endpoint = keyword
      ? `/api/channel/search?keyword=${encodeURIComponent(keyword)}`
      : '/api/channel/?scope=all';
    const res = await API.get(endpoint);
    const { success, message, data } = res.data;
    if (success) {
      setChannels(data.map(processChannelData));
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const refresh = async () => {
    setLoading(true);
    await loadChannels(searchKeyword);
  };

  const toggleShowDetail = () => {
    setShowDetail(!showDetail);
    localStorage.setItem('show_detail', (!showDetail).toString());
  };

  useEffect(() => {
    loadChannels()
      .then()
      .catch((reason) => {
        showError(reason);
      });
    loadChannelModels().then();
  }, []);

  const manageChannel = async (id, action, idx, value) => {
    let data = { id };
    let res;
    switch (action) {
      case 'delete':
        res = await API.delete(`/api/channel/${id}/`);
        break;
      case 'enable':
        data.status = 1;
        res = await API.put('/api/channel/', data);
        break;
      case 'disable':
        data.status = 2;
        res = await API.put('/api/channel/', data);
        break;
      case 'priority':
        if (value === '') {
          return;
        }
        data.priority = parseInt(value);
        res = await API.put('/api/channel/', data);
        break;
      case 'weight':
        if (value === '') {
          return;
        }
        data.weight = parseInt(value);
        if (data.weight < 0) {
          data.weight = 0;
        }
        res = await API.put('/api/channel/', data);
        break;
      default:
        return;
    }
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('channel.messages.operation_success'));
      let channel = res.data.data;
      let newChannels = [...channels];
      if (action === 'delete') {
        newChannels[idx].deleted = true;
      } else if (channel) {
        newChannels[idx] = processChannelData({
          ...newChannels[idx],
          ...channel,
        });
      }
      setChannels(newChannels);
    } else {
      showError(message);
    }
  };

  const reorderChannelV2 = async (id, direction) => {
    const actionKey = `${id}-${direction}`;
    setReorderingChannelAction(actionKey);
    const res = await API.post('/api/channel/reorder', { id, direction });
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('channel.messages.operation_success'));
      await refresh();
    } else {
      showError(message);
    }
    setReorderingChannelAction('');
  };

  const renderStatus = (channel, t) => {
    let status = channel.status;
    let failures = channel.failures || 0;
    switch (status) {
      case 1:
        if (failures >= 4) {
          return (
            <Popup
              trigger={
                <Label basic color='red'>
                  失效 ({failures})
                </Label>
              }
              content="最近请求失败过多"
              basic
            />
          );
        } else if (failures > 0) {
          return (
            <Popup
              trigger={
                <Label basic color='orange'>
                  不稳定 ({failures})
                </Label>
              }
              content="最近有请求失败"
              basic
            />
          );
        }
        return (
          <Label basic color='green'>
            {t('channel.table.status_enabled')}
          </Label>
        );
      case 2:
        return (
          <Popup
            trigger={
              <Label basic color='red'>
                {t('channel.table.status_disabled')}
              </Label>
            }
            content={t('channel.table.status_disabled_tip')}
            basic
          />
        );
      case 3:
        return (
          <Popup
            trigger={
              <Label basic color='red'>
                {t('channel.table.status_auto_disabled')}
              </Label>
            }
            content={t('channel.table.status_auto_disabled_tip')}
            basic
          />
        );
      default:
        return (
          <Label basic color='grey'>
            {t('channel.table.status_unknown')}
          </Label>
        );
    }
  };

  const renderResponseTime = (responseTime, t) => {
    let time = responseTime / 1000;
    time = time.toFixed(2) + 's';
    if (responseTime === 0) {
      return (
        <Label basic color='grey'>
          {t('channel.table.not_tested')}
        </Label>
      );
    } else if (responseTime <= 1000) {
      return (
        <Label basic color='green'>
          {time}
        </Label>
      );
    } else if (responseTime <= 3000) {
      return (
        <Label basic color='olive'>
          {time}
        </Label>
      );
    } else if (responseTime <= 5000) {
      return (
        <Label basic color='yellow'>
          {time}
        </Label>
      );
    } else {
      return (
        <Label basic color='red'>
          {time}
        </Label>
      );
    }
  };

  const searchChannels = async () => {
    setSearching(true);
    setLoading(true);
    await loadChannels(searchKeyword);
    setSearching(false);
  };

  const switchTestModel = async (idx, model) => {
    let newChannels = [...channels];
    newChannels[idx].test_model = model;
    setChannels(newChannels);
  };

  const testChannel = async (id, name, idx, m) => {
    const res = await API.get(`/api/channel/test/${id}?model=${m}`);
    const { success, message, time, model } = res.data;
    if (success) {
      let newChannels = [...channels];
      newChannels[idx].response_time = time * 1000;
      newChannels[idx].test_time = Date.now() / 1000;
      setChannels(newChannels);
      showSuccess(
        t('channel.messages.test_success', { name, model, time, message })
      );
    } else {
      showError(message);
    }
    let newChannels = [...channels];
    newChannels[idx].response_time = time * 1000;
    newChannels[idx].test_time = Date.now() / 1000;
    setChannels(newChannels);
  };

  const testChannels = async (scope) => {
    const res = await API.get(`/api/channel/test?scope=${scope}`);
    const { success, message } = res.data;
    if (success) {
      showInfo(t('channel.messages.test_all_started'));
    } else {
      showError(message);
    }
  };

  const deleteAllDisabledChannels = async () => {
    const res = await API.delete(`/api/channel/disabled`);
    const { success, message, data } = res.data;
    if (success) {
      showSuccess(
        t('channel.messages.delete_disabled_success', { count: data })
      );
      await refresh();
    } else {
      showError(message);
    }
  };

  const updateChannelBalance = async (id, name, idx) => {
    const res = await API.get(`/api/channel/update_balance/${id}/`);
    const { success, message, balance } = res.data;
    if (success) {
      let newChannels = [...channels];
      newChannels[idx].balance = balance;
      newChannels[idx].balance_updated_time = Date.now() / 1000;
      setChannels(newChannels);
      showSuccess(t('channel.messages.balance_update_success', { name }));
    } else {
      showError(message);
    }
  };

  const updateAllChannelsBalance = async () => {
    setUpdatingBalance(true);
    const res = await API.get(`/api/channel/update_balance`);
    const { success, message } = res.data;
    if (success) {
      showInfo(t('channel.messages.all_balance_updated'));
    } else {
      showError(message);
    }
    setUpdatingBalance(false);
  };

  const handleKeywordChange = async (e, { value }) => {
    setSearchKeyword(value.trim());
  };

  const sortChannel = (key) => {
    if (channels.length === 0) return;
    setLoading(true);
    let sortedChannels = [...channels];
    sortedChannels.sort((a, b) => {
      if (!isNaN(a[key])) {
        return a[key] - b[key];
      } else {
        return ('' + a[key]).localeCompare(b[key]);
      }
    });
    if (sortedChannels[0].id === channels[0].id) {
      sortedChannels.reverse();
    }
    setChannels(sortedChannels);
    setLoading(false);
  };

  const visibleChannels = channels.filter((channel) => !channel.deleted);

  return (
    <>
      <Form onSubmit={searchChannels}>
        <Form.Input
          icon='search'
          fluid
          iconPosition='left'
          placeholder={t('channel.search')}
          value={searchKeyword}
          loading={searching}
          onChange={handleKeywordChange}
        />
      </Form>
      {showPrompt && (
        <Message
          onDismiss={() => {
            setShowPrompt(false);
            setPromptShown(promptID);
          }}
        >
          {t('channel.balance_notice')}
          <br />
          {t('channel.test_notice')}
          <br />
          {t('channel.detail_notice')}
        </Message>
      )}
      <Table basic={'very'} compact size='small'>
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('id');
              }}
            >
              {t('channel.table.id')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('name');
              }}
            >
              {t('channel.table.name')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('group');
              }}
            >
              {t('channel.table.group')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('type');
              }}
            >
              {t('channel.table.type')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('status');
              }}
            >
              {t('channel.table.status')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('response_time');
              }}
            >
              {t('channel.table.response_time')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('balance');
              }}
            >
              {t('channel.table.balance')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortChannel('priority');
              }}
              hidden={!showDetail}
            >
              {t('channel.table.priority')}
            </Table.HeaderCell>
            <Table.HeaderCell>
              {t('channel.table.management_order')}
            </Table.HeaderCell>
            <Table.HeaderCell hidden={!showDetail}>
              {t('channel.table.test_model')}
            </Table.HeaderCell>
            <Table.HeaderCell>{t('channel.table.actions')}</Table.HeaderCell>
          </Table.Row>
        </Table.Header>

        <Table.Body>
          {visibleChannels.map((channel, idx) => {
            return (
              <Table.Row key={channel.id}>
                <Table.Cell>{channel.id}</Table.Cell>
                <Table.Cell>
                  {channel.name ? channel.name : t('channel.table.no_name')}
                  {channel.has_disabled_keys && (
                    <Label color='red' size='tiny' style={{ marginLeft: '4px' }}>
                      有失效的密钥
                    </Label>
                  )}
                </Table.Cell>
                <Table.Cell>{renderGroup(channel.group)}</Table.Cell>
                <Table.Cell>{renderType(channel.type, t)}</Table.Cell>
                <Table.Cell>{renderStatus(channel, t)}</Table.Cell>
                <Table.Cell>
                  <Popup
                    content={
                      channel.test_time
                        ? renderTimestamp(channel.test_time)
                        : t('channel.table.not_tested')
                    }
                    key={channel.id}
                    trigger={renderResponseTime(channel.response_time, t)}
                    basic
                  />
                </Table.Cell>
                <Table.Cell>
                  <Popup
                    trigger={
                      <span
                        onClick={() => {
                          updateChannelBalance(channel.id, channel.name, idx);
                        }}
                        style={{ cursor: 'pointer' }}
                      >
                        {renderBalance(channel.type, channel.balance, t)}
                      </span>
                    }
                    content={t('channel.table.click_to_update')}
                    basic
                  />
                </Table.Cell>
                <Table.Cell hidden={!showDetail}>
                  <Popup
                    trigger={
                      <Input
                        type='number'
                        defaultValue={channel.priority}
                        onBlur={(event) => {
                          manageChannel(
                            channel.id,
                            'priority',
                            idx,
                            event.target.value
                          );
                        }}
                      >
                        <input style={{ maxWidth: '60px' }} />
                      </Input>
                    }
                    content={t('channel.table.priority_tip')}
                    basic
                  />
                </Table.Cell>
                <Table.Cell>
                  <Popup
                    trigger={
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          flexWrap: 'wrap',
                          gap: '4px',
                        }}
                      >
                        <Label basic>{channel.display_order || '-'}</Label>
                        <Button
                          size='tiny'
                          type='button'
                          loading={reorderingChannelAction === `${channel.id}-up`}
                          onClick={() => {
                            reorderChannelV2(channel.id, 'up');
                          }}
                        >
                          {t('channel.buttons.move_up')}
                        </Button>
                        <Button
                          size='tiny'
                          type='button'
                          loading={reorderingChannelAction === `${channel.id}-down`}
                          onClick={() => {
                            reorderChannelV2(channel.id, 'down');
                          }}
                        >
                          {t('channel.buttons.move_down')}
                        </Button>
                      </div>
                    }
                    content={t('channel.table.management_order_tip')}
                    basic
                  />
                </Table.Cell>
                <Table.Cell hidden={!showDetail}>
                  <Dropdown
                    placeholder={t('channel.table.select_test_model')}
                    selection
                    options={channel.model_options}
                    defaultValue={channel.test_model}
                    onChange={(event, data) => {
                      switchTestModel(idx, data.value);
                    }}
                  />
                </Table.Cell>
                <Table.Cell>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      flexWrap: 'wrap',
                      gap: '2px',
                      rowGap: '6px',
                    }}
                  >
                    <Button
                      size={'tiny'}
                      positive
                      onClick={() => {
                        testChannel(
                          channel.id,
                          channel.name,
                          idx,
                          channel.test_model
                        );
                      }}
                    >
                      {t('channel.buttons.test')}
                    </Button>
                    <Popup
                      trigger={
                        <Button size='tiny' negative>
                          {t('channel.buttons.delete')}
                        </Button>
                      }
                      on='click'
                      flowing
                      hoverable
                    >
                      <Button
                        size={'tiny'}
                        negative
                        onClick={() => {
                          manageChannel(channel.id, 'delete', idx);
                        }}
                      >
                        {t('channel.buttons.confirm_delete')} {channel.name}
                      </Button>
                    </Popup>
                    <Button
                      size={'tiny'}
                      onClick={() => {
                        manageChannel(
                          channel.id,
                          channel.status === 1 ? 'disable' : 'enable',
                          idx
                        );
                      }}
                    >
                      {channel.status === 1
                        ? t('channel.buttons.disable')
                        : t('channel.buttons.enable')}
                    </Button>
                    <Button
                      size={'tiny'}
                      as={Link}
                      to={'/channel/edit/' + channel.id}
                    >
                      {t('channel.buttons.edit')}
                    </Button>
                  </div>
                </Table.Cell>
              </Table.Row>
            );
          })}
        </Table.Body>

        <Table.Footer>
          <Table.Row>
            <Table.HeaderCell colSpan={showDetail ? '11' : '9'}>
              <Button size='tiny' as={Link} to='/channel/add' loading={loading}>
                {t('channel.buttons.add')}
              </Button>
              <Button
                size='tiny'
                loading={loading}
                onClick={() => {
                  testChannels('all');
                }}
              >
                {t('channel.buttons.test_all')}
              </Button>
              <Button
                size='tiny'
                loading={loading}
                onClick={() => {
                  testChannels('disabled');
                }}
              >
                {t('channel.buttons.test_disabled')}
              </Button>
              <Popup
                trigger={
                  <Button size='tiny' loading={loading}>
                    {t('channel.buttons.delete_disabled')}
                  </Button>
                }
                on='click'
                flowing
                hoverable
              >
                <Button
                  size='tiny'
                  loading={loading}
                  negative
                  onClick={deleteAllDisabledChannels}
                >
                  {t('channel.buttons.confirm_delete_disabled')}
                </Button>
              </Popup>
              <Button
                size='tiny'
                onClick={updateAllChannelsBalance}
                loading={updatingBalance}
              >
                {t('channel.buttons.refresh_balance')}
              </Button>
              <Button size='tiny' onClick={refresh} loading={loading}>
                {t('channel.buttons.refresh')}
              </Button>
              <Button size='tiny' onClick={toggleShowDetail}>
                {showDetail
                  ? t('channel.buttons.hide_detail')
                  : t('channel.buttons.show_detail')}
              </Button>
              <span style={{ float: 'right', paddingTop: '8px' }}>
                {t('channel.table.total', { count: visibleChannels.length })}
              </span>
            </Table.HeaderCell>
          </Table.Row>
        </Table.Footer>
      </Table>
    </>
  );
};

export default ChannelsTable;
