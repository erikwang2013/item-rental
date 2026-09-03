import 'package:flutter/material.dart';

import '../../api/endpoints.dart';
import '../../models/item.dart';
import '../../widgets/commons.dart';
import 'list.dart';

/// 搜索页:关键词 + city(可选),调 GET /items/search(q=...)。
class SearchPage extends StatefulWidget {
  const SearchPage({super.key});

  @override
  State<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends State<SearchPage> {
  final _kw = TextEditingController();
  final _city = TextEditingController();
  final List<Item> _results = [];
  bool _loading = false;
  bool _searched = false;

  @override
  void dispose() {
    _kw.dispose();
    _city.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final q = _kw.text.trim();
    if (q.isEmpty) return toast('请输入关键词');
    FocusScope.of(context).unfocus();
    setState(() { _loading = true; _searched = true; });
    try {
      final city = _city.text.trim();
      final d = await api.searchItems(q: q, city: city.isEmpty ? null : city);
      setState(() => _results
        ..clear()
        ..addAll(parseItemList(d)));
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: kGreen,
        foregroundColor: Colors.white,
        title: TextField(
          controller: _kw,
          autofocus: true,
          textInputAction: TextInputAction.search,
          onSubmitted: (_) => _search(),
          style: const TextStyle(color: Colors.white),
          cursorColor: Colors.white,
          decoration: const InputDecoration(
            hintText: '搜索关键词',
            hintStyle: TextStyle(color: Colors.white70),
            border: InputBorder.none,
          ),
        ),
        actions: [TextButton(onPressed: _search, child: const Text('搜索', style: TextStyle(color: Colors.white)))],
      ),
      body: Column(children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
          child: TextField(
            controller: _city,
            decoration: const InputDecoration(
                labelText: '城市(可选)',
                prefixIcon: Icon(Icons.location_on_outlined),
                isDense: true),
            onSubmitted: (_) => _search(),
          ),
        ),
        Expanded(
          child: _loading
              ? const StatusBox(loading: true)
              : !_searched
                  ? const StatusBox(emptyText: '输入关键词搜索物品')
                  : _results.isEmpty
                      ? const StatusBox(emptyText: '没有找到相关物品')
                      : ListView.builder(
                          padding: const EdgeInsets.all(8),
                          itemCount: _results.length,
                          itemBuilder: (_, i) =>
                              ItemListTile(item: _results[i]),
                        ),
        ),
      ]),
    );
  }
}
