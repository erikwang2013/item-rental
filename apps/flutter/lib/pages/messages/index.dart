import 'package:flutter/material.dart';

import '../../api/endpoints.dart';
import '../../models/message.dart';
import '../../widgets/commons.dart';

/// 消息中心:最新优先;点未读条目即标记已读。
class MessagesPage extends StatefulWidget {
  const MessagesPage({super.key});

  @override
  State<MessagesPage> createState() => _MessagesPageState();
}

class _MessagesPageState extends State<MessagesPage> {
  final List<Msg> _msgs = [];
  int _page = 1;
  int _total = 0;
  bool _loading = true;
  bool _ended = false;
  bool _unreadOnly = false;
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    _scroll.addListener(() {
      if (_scroll.position.pixels > _scroll.position.maxScrollExtent - 150) _more();
    });
    _reload();
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    setState(() { _page = 1; _ended = false; });
    await _fetch(1, clear: true);
  }

  void _more() {
    if (!_loading && !_ended && _msgs.length >= _total && _total > 0) {
      setState(() => _ended = true);
      return;
    }
    if (!_loading && !_ended) _fetch(_page + 1);
  }

  Future<void> _fetch(int page, {bool clear = false}) async {
    setState(() => _loading = true);
    try {
      final d = await api.messages(unreadOnly: _unreadOnly, page: page);
      final list = (d['messages'] as List<dynamic>? ?? [])
          .whereType<Map<String, dynamic>>()
          .map(Msg.fromJson)
          .toList();
      setState(() {
        _total = d['total'] as int? ?? 0;
        _page = page;
        if (clear) _msgs.clear();
        _msgs.addAll(list);
        _ended = _msgs.length >= _total;
      });
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _open(Msg m) async {
    if (m.read) return; // 内容已展示在列表中
    try {
      await api.markRead(m.id);
    } catch (_) {}
    if (mounted) {
      setState(() {
        for (final x in _msgs) {
          if (x.id == m.id) x.read = true;
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('消息中心'),
        actions: [
          TextButton(
            onPressed: () {
              setState(() => _unreadOnly = !_unreadOnly);
              _reload();
            },
            child: Text(_unreadOnly ? '全部' : '只看未读',
                style: TextStyle(color: _unreadOnly ? kGreen : Colors.grey)),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _reload,
        child: _msgs.isEmpty
            ? ListView(children: [
                const SizedBox(height: 200),
                StatusBox(loading: _loading, emptyText: '暂无消息')
              ])
            : ListView.builder(
                controller: _scroll,
                itemCount: _msgs.length + 1,
                itemBuilder: (ctx, i) {
                  if (i == _msgs.length) {
                    return Padding(
                      padding: const EdgeInsets.all(12),
                      child: Center(
                        child: _ended
                            ? Text('已全部加载', style: TextStyle(fontSize: 12, color: Colors.grey.shade500))
                            : (_loading ? const CircularProgressIndicator() : const SizedBox()),
                      ),
                    );
                  }
                  final m = _msgs[i];
                  return ListTile(
                    leading: Icon(
                      Icons.notifications_active_outlined,
                      color: m.read ? Colors.grey.shade400 : kGreen,
                    ),
                    title: Row(children: [
                      Expanded(
                          child: Text(m.title,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                  fontWeight: m.read ? FontWeight.normal : FontWeight.bold))),
                      if (!m.read)
                        Container(
                          width: 8,
                          height: 8,
                          margin: const EdgeInsets.only(left: 6),
                          decoration: const BoxDecoration(color: Colors.red, shape: BoxShape.circle),
                        ),
                    ]),
                    subtitle: Text(
                      '${m.typeText} · ${m.content}',
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontSize: 12),
                    ),
                    onTap: () => _open(m),
                  );
                },
              ),
      ),
    );
  }
}
