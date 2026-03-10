/* ./src/js/app.js */
document.addEventListener('DOMContentLoaded', function () {
    // 获取DOM元素
    const searchInput = document.getElementById('home-steam-search');
    const searchBtn = document.getElementById('home-steam-search-btn');
    const featuredGamesContainer = document.getElementById('featured-games');
    const errorDiv = document.getElementById('featured-error');
    const libraryList = document.getElementById('library-list');
    const refreshLibraryBtn = document.getElementById('refresh-library');
    let addedLibrary = new Map();
    let gamesLoaded = false;

    async function loadAddedLibrary() {
        try {
            if (!window.go?.main?.App?.GetAddedLibraries) return;
            const response = await window.go.main.App.GetAddedLibraries();
            const data = typeof response === 'string' ? JSON.parse(response || '[]') : (response || []);

            addedLibrary = new Map();
            if (Array.isArray(data)) {
                data.forEach(item => {
                    if (item && item.appid) {
                        addedLibrary.set(String(item.appid), item);
                    }
                });
            }

            renderLibraryList();
            syncDisplayedGameItems();
        } catch (error) {
            console.error('读取入库记录失败:', error);
        }
    }

    function renderLibraryList() {
        if (!libraryList) return;
        if (!addedLibrary.size) {
            libraryList.innerHTML = '<p class="text-center">暂无入库记录</p>';
            return;
        }

        const items = Array.from(addedLibrary.values()).sort((a, b) => {
            const at = a.added_at || '';
            const bt = b.added_at || '';
            return bt.localeCompare(at);
        });

        libraryList.innerHTML = items.map(item => {
            const name = item.name || `AppID: ${item.appid}`;
            const timeText = item.added_at ? new Date(item.added_at).toLocaleString() : '';
            return `
            <div class="library-item card">
                <div class="library-info">
                    <div class="fw-bold">${name}</div>
                    <div class="small">AppID: ${item.appid}</div>
                    <div class="small">${timeText}</div>
                </div>
                <div class="library-actions">
                    <button class="app-btn app-btn-sm library-remove-btn" data-appid="${item.appid}">
                        <i class="fa fa-trash me-1"></i>移除
                    </button>
                </div>
            </div>`;
        }).join('');
    }

    function setGameItemState(gameItem, isAdded) {
        if (!gameItem) return;
        const appId = gameItem.dataset.appid;
        const actions = gameItem.querySelector('.game-actions');
        if (!actions || !appId) return;

        if (isAdded) {
            actions.innerHTML = `
                <span class="status-badge">�����</span>
                <button class="app-btn app-btn-sm remove-from-library-btn" data-appid="${appId}">
                    <i class="fa fa-trash me-1"></i>�Ƴ�
                </button>`;
        } else {
            actions.innerHTML = `
                <button class="app-btn app-btn-sm add-to-library-btn" data-appid="${appId}">
                    <i class="fa fa-download me-1"></i>入库
                </button>`;
        }
    }

    function syncDisplayedGameItems() {
        document.querySelectorAll('.game-item').forEach(item => {
            const appId = item.dataset.appid;
            setGameItemState(item, addedLibrary.has(String(appId)));
        });
    }

    // 从Wails后端获取Steam特色游戏列表
    async function loadFeaturedGames() {
        try {
            errorDiv.textContent = '';
            featuredGamesContainer.innerHTML = '<div class="text-center p-3">加载�?..</div>';

            // 检查Wails函数是否存在并调�?
            if (!window.go?.main?.App?.GetSteamFeatured) {
                throw new Error('Wails backend unavailable. Please restart the app.');
            }

            const response = await window.go.main.App.GetSteamFeatured();
            // 检查响应是否为有效�?
            if (!response) {
                throw new Error('从后端获取的数据为空: ' + response);
            }
            const data = typeof response === 'string' ? JSON.parse(response) : response;
            displayGames(data);
            gamesLoaded = true;
        } catch (error) {
            errorDiv.textContent = error.toString();
            featuredGamesContainer.innerHTML = '';
            gamesLoaded = false;
        }
    }

    // 显示游戏数据
    function displayGames(data) {
        featuredGamesContainer.innerHTML = '';

        // 初始化数�?
        let games = [];

        // 处理搜索API返回格式 (items数组)
        if (data && data.items && Array.isArray(data.items)) {
            games = data.items;
        }
        // 处理特色游戏API返回格式
        else if (data && (data.featured_win || data.featured_mac || data.featured_linux)) {
            const platformGames = [];
            if (Array.isArray(data.featured_win)) platformGames.push(...data.featured_win);
            if (Array.isArray(data.featured_mac)) platformGames.push(...data.featured_mac);
            if (Array.isArray(data.featured_linux)) platformGames.push(...data.featured_linux);
            games = platformGames;
        }

        if (games.length === 0) {
            featuredGamesContainer.innerHTML = '<p class="text-center">没有找到游戏数据</p>';
            return;
        }

        // 去重
        const uniqueGames = [];
        const appIdSet = new Set();
        for (const game of games) {
            const appId = game.id || game.appid;
            if (appId && !appIdSet.has(appId)) {
                appIdSet.add(appId);
                uniqueGames.push(game);
            }
        }

        // 创建文档片段以提高性能
        const fragment = document.createDocumentFragment();

        for (const game of uniqueGames) {
            const gameElement = document.createElement('div');
            gameElement.className = 'game-item';

            // 适配Steam Store API字段
            const appId = game.id || game.appid;
            gameElement.dataset.appid = String(appId);
            const gameName = game.name || game.title || '未知游戏';

            // 图片URL处理
            let imageSrc = '';
            if (game.header_image) {
                imageSrc = game.header_image;
            } else if (game.tiny_image) {
                imageSrc = game.tiny_image;
            }

            // 价格信息处理
            let priceInfo = '';

            // 储存两种 API 不同的字�?
            const finalPrice = (game.price && game.price.final) || game.final_price || 0;
            const initialPrice = (game.price && game.price.initial) || game.original_price || finalPrice;
            const discountPercent = game.discount_percent || (initialPrice > finalPrice ?
                Math.round((1 - finalPrice / initialPrice) * 100) : 0);

            if (finalPrice === 0 || game.is_free) {
                priceInfo = '免费';
            } else if (finalPrice < initialPrice) {
                priceInfo = `¥ ${(finalPrice / 100).toFixed(2)} <span class="text-danger">(-${discountPercent}%)</span> <span class="text-decoration-line-through">¥${(initialPrice / 100).toFixed(2)}</span>`;
            } else {
                priceInfo = `¥ ${(finalPrice / 100).toFixed(2)}`;
            }

                        const isAdded = addedLibrary.has(String(appId));
            const actionHtml = isAdded
                ? `<span class="status-badge">�����</span>
                   <button class="app-btn app-btn-sm remove-from-library-btn" data-appid="${appId}">
                     <i class="fa fa-trash me-1"></i>�Ƴ�
                   </button>`
                : `<button class="app-btn app-btn-sm add-to-library-btn" data-appid="${appId}">
                     <i class="fa fa-download me-1"></i>���
                   </button>`;

            gameElement.innerHTML = `
            <div class="game-info flex-row justify-content-between align-items-center card">
                <div class="d-flex align-items-center">
                    <img src="${imageSrc}" class="card-img me-3">
                    <div>
                        <div class="fw-bold"><a class="game-name" href="https://store.steampowered.com/app/${appId}/" target="_blank" rel="noopener noreferrer">${gameName}</a></div>
                        <span class="small">AppID: ${appId}</span>
                        <div>${priceInfo}</div>
                    </div>
                </div>
                <div class="game-actions">
                    ${actionHtml}
                </div>
            </div>`;
            fragment.appendChild(gameElement);
        }

        featuredGamesContainer.innerHTML = '';
        featuredGamesContainer.appendChild(fragment);

        // 处理入库按钮点击
        // 改造入库按钮点击事�?
        featuredGamesContainer.addEventListener('click', async function (event) {
            const addBtn = event.target.closest('.add-to-library-btn');
            const removeBtn = event.target.closest('.remove-from-library-btn');
            if (!addBtn && !removeBtn) return;

            const btn = addBtn || removeBtn;
            if (btn.dataset.loading === 'true') return;

            const appID = btn.getAttribute('data-appid');
            const gameItem = btn.closest('.game-item');
            const gameName = gameItem && gameItem.querySelector('.game-name')
                ? gameItem.querySelector('.game-name').textContent.trim()
                : '';
            const originalHTML = btn.innerHTML;
            let succeeded = false;

            btn.dataset.loading = 'true';
            btn.innerHTML = addBtn ? '<i class="fa fa-spinner fa-spin me-1"></i>�����...' : '<i class="fa fa-spinner fa-spin me-1"></i>�Ƴ���...';
            btn.disabled = true;

            try {
                if (addBtn) {
                    updateProgress(0);
                    const result = await window.go.main.App.AddGameToLibrary(appID, gameName);
                    updateProgress(100);
                    showToasts(result, 'success');
                    succeeded = true;
                    await loadAddedLibrary();
                    setGameItemState(gameItem, true);
                } else {
                    const result = await window.go.main.App.RemoveGameFromLibrary(appID);
                    showToasts(result, 'success');
                    succeeded = true;
                    await loadAddedLibrary();
                    setGameItemState(gameItem, false);
                }
            } catch (error) {
                showToasts(error.toString(), 'error');
                hideProgress();
            } finally {
                if (!succeeded) {
                    btn.innerHTML = originalHTML;
                    btn.disabled = false;
                    btn.dataset.loading = 'false';
                }
            }
        });
    }

    // 搜索游戏
    async function searchGames(searchTerm) {
        try {
            errorDiv.textContent = '';
            featuredGamesContainer.innerHTML = `<div class="text-center p-3">搜索�?..</div>`;

            // 检查Wails搜索函数是否存在
            if (!window.go?.main?.App?.SearchSteamGames) {
                throw new Error('Wails backend unavailable. Please restart the app.');
            }

            // 调用后端搜索接口
            const response = await window.go.main.App.SearchSteamGames(searchTerm);
            if (!response) {
                throw new Error('未找到匹配的游戏');
            }
            const data = typeof response === 'string' ? JSON.parse(response) : response;

            // 适配Steam Store API搜索返回格式
            displayGames(data);
        } catch (error) {
            errorDiv.textContent = error.toString();
            featuredGamesContainer.innerHTML = '';
        }
    }

    // 搜索按钮事件
    searchBtn.addEventListener('click', function () {
        const searchTerm = searchInput.value.trim();
        if (searchTerm) {
            searchGames(searchTerm);
        } else {
            // 搜索框为空时，重新加载热门游�?
            loadFeaturedGames();
        }
    });

    // 支持按回车键搜索
    searchInput.addEventListener('keypress', function (event) {
        if (event.key === 'Enter') {
            searchBtn.click();
        }
    });

    // 页面加载完成后自动加载游戏列�?仅首�?
    if (document.getElementById('home-page') && !document.getElementById('home-page').classList.contains('d-none') && !gamesLoaded) {
        loadFeaturedGames();
    }

    // 监听页面切换事件，仅在首次进入home页面时加载游�?
    document.querySelectorAll('.nav-item').forEach(navItem => {
        navItem.addEventListener('click', function () {
            if (this.getAttribute('data-page') === 'home' && !gamesLoaded) {
                setTimeout(loadFeaturedGames, 100);
            }
        });
    });
    if (libraryList) {
        libraryList.addEventListener('click', async function (event) {
            const btn = event.target.closest('.library-remove-btn');
            if (!btn || btn.dataset.loading === 'true') return;

            const appID = btn.getAttribute('data-appid');
            const originalHTML = btn.innerHTML;
            btn.dataset.loading = 'true';
            btn.disabled = true;
            btn.innerHTML = '<i class="fa fa-spinner fa-spin me-1"></i>�Ƴ���...';

            try {
                const result = await window.go.main.App.RemoveGameFromLibrary(appID);
                showToasts(result, 'success');
                await loadAddedLibrary();
            } catch (error) {
                showToasts(error.toString(), 'error');
                btn.innerHTML = originalHTML;
                btn.disabled = false;
                btn.dataset.loading = 'false';
            }
        });
    }

    if (refreshLibraryBtn) {
        refreshLibraryBtn.addEventListener('click', function () {
            loadAddedLibrary();
        });
    }

    loadAddedLibrary();
});



